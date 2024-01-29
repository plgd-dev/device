/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	bridgeNet "github.com/plgd-dev/device/v2/bridge/net"
	ocfCloud "github.com/plgd-dev/device/v2/pkg/ocf/cloud"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	coapTcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	coapTcpServer "github.com/plgd-dev/go-coap/v3/tcp/server"
)

// Service is a configuration of coap-gateway
type Service struct {
	coapServer *coapTcpServer.Server
	listener   coapTcpServer.Listener
	closeFn    []func()
	ctx        context.Context
	cancel     context.CancelFunc
	sigs       chan os.Signal
	getHandler GetServiceHandler
	clients    []*Client
}

func newListener(cfg Config) (coapTcpServer.Listener, func(), error) {
	if !cfg.TLS.Enabled {
		listener, err := net.NewTCPListener("tcp", cfg.Addr)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot create tcp listener: %w", err)
		}
		closeListener := func() {
			if errC := listener.Close(); errC != nil {
				fmt.Printf("failed to close tcp listener: %v\n", errC)
			}
		}
		return listener, closeListener, nil
	}

	listener, err := net.NewTLSListener("tcp", cfg.Addr, cfg.TLS.Config)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create tcp-tls listener: %w", err)
	}
	closeFn := (func() {
		if errC := listener.Close(); errC != nil {
			fmt.Printf("failed to close tcp-tls listener: %v\n", errC)
		}
	})
	return listener, closeFn, nil
}

// New creates server.
func New(ctx context.Context, cfg Config, getHandler GetServiceHandler) (*Service, error) {
	var closeFn []func()
	listener, closeListener, err := newListener(cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create listener: %w", err)
	}
	closeFn = append(closeFn, closeListener)

	ctx, cancel := context.WithCancel(ctx)

	s := Service{
		listener:   listener,
		closeFn:    closeFn,
		ctx:        ctx,
		cancel:     cancel,
		sigs:       make(chan os.Signal, 1),
		getHandler: getHandler,
		clients:    nil,
	}

	if err := s.setupCoapServer(); err != nil {
		return nil, fmt.Errorf("cannot setup coap server: %w", err)
	}

	return &s, nil
}

const clientKey = "client"

func (s *Service) coapConnOnNew(coapConn *coapTcpClient.Conn) {
	client := newClient(s, coapConn, s.getHandler(s, WithCoapConnectionOpt(coapConn)))
	coapConn.SetContextValue(clientKey, client)
	coapConn.AddOnClose(func() {
		client.OnClose()
	})
	s.clients = append(s.clients, client)
}

func validateCommand(writer mux.ResponseWriter, request *mux.Message, server *Service, fnc func(req *mux.Message, client *Client)) {
	request.Hijack()
	go func(w mux.ResponseWriter, req *mux.Message) {
		client, ok := w.Conn().Context().Value(clientKey).(*Client)
		if !ok || client == nil {
			con, ok2 := w.Conn().(*coapTcpClient.Conn)
			if !ok2 {
				panic("invalid connection")
			}
			client = newClient(server, con, nil)
		}
		closeClient := func(c *Client) {
			if err := c.Close(); err != nil {
				fmt.Printf("cannot handle command: %v\n", err)
			}
		}

		switch req.Code() {
		case codes.POST, codes.DELETE, codes.PUT, codes.GET:
			fnc(req, client)
		case codes.Empty:
			if !ok {
				client.sendErrorResponse(fmt.Errorf("cannot handle command: client not found"), codes.InternalServerError, req.Token())
				closeClient(client)
				return
			}
		case codes.Content:
			// Unregistered observer at a peer send us a notification
		default:
			fmt.Printf("received invalid code: CoapCode(%v)", req.Code())
		}
	}(writer, request)
}

func defaultHandler(req *mux.Message, client *Client) {
	path, _ := req.Options().Path()
	client.sendErrorResponse(fmt.Errorf("DeviceId: %v: unknown path %v", client.GetDeviceID(), path), codes.NotFound, req.Token())
}

func (s *Service) setupCoapServer() error {
	setHandlerError := func(uri string, err error) error {
		return fmt.Errorf("failed to set %v handler: %w", uri, err)
	}
	m := mux.NewRouter()
	m.DefaultHandle(mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, s, defaultHandler)
	}))
	if err := m.Handle(ocfCloud.SignUp, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, s, signUpHandler)
	})); err != nil {
		return setHandlerError(ocfCloud.SignUp, err)
	}
	if err := m.Handle(ocfCloud.SignIn, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, s, signInHandler)
	})); err != nil {
		return setHandlerError(ocfCloud.SignIn, err)
	}
	if err := m.Handle(ocfCloud.ResourceDirectory, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, s, resourceDirectoryHandler)
	})); err != nil {
		return setHandlerError(ocfCloud.ResourceDirectory, err)
	}
	if err := m.Handle(ocfCloud.RefreshToken, mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		validateCommand(w, r, s, refreshTokenHandler)
	})); err != nil {
		return setHandlerError(ocfCloud.RefreshToken, err)
	}

	opts := make([]coapTcpServer.Option, 0, 5)
	opts = append(opts, options.WithOnNewConn(s.coapConnOnNew))
	opts = append(opts, options.WithMux(m))
	opts = append(opts, options.WithContext(s.ctx))
	opts = append(opts, options.WithErrors(func(e error) {
		fmt.Printf("test-coap: %v\n", e)
	}))
	opts = append(opts, options.WithMaxMessageSize(bridgeNet.DefaultMaxMessageSize))
	s.coapServer = tcp.NewServer(opts...)
	return nil
}

func (s *Service) Serve() error {
	return s.serveWithHandlingSignal()
}

func (s *Service) serveWithHandlingSignal() error {
	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	go func(server *Service) {
		defer wg.Done()
		err = server.coapServer.Serve(server.listener)
		server.cancel()
		for i := range server.closeFn {
			server.closeFn[len(server.closeFn)-1-i]()
		}
	}(s)

	signal.Notify(s.sigs,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-s.sigs

	s.coapServer.Stop()
	wg.Wait()

	return err
}

func (s *Service) GetClients() []*Client {
	return s.clients
}

// Close turns off the server.
func (s *Service) Close() error {
	select {
	case s.sigs <- syscall.SIGTERM:
	default:
	}
	return nil
}
