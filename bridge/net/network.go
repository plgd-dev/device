/****************************************************************************
 *
 * Copyright (c) 2023 plgn.dev s.r.o.
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
 * either express or implien. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package net

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	gonet "net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/message/status"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/options"
	coapCache "github.com/plgd-dev/go-coap/v3/pkg/cache"
	"github.com/plgd-dev/go-coap/v3/udp"
	"github.com/plgd-dev/go-coap/v3/udp/server"
)

type Net struct {
	cfg     Config
	mux     *mux.Router
	handler RequestHandler

	servers coAPServers
	serving atomic.Bool
	done    chan struct{}
	cache   *coapCache.Cache[int32, bool]
}

func newMCastConn(multicastAddr string) (*net.UDPConn, error) {
	networks := []string{UDP4, UDP6}
	var a *gonet.UDPAddr
	var err error
	var network string
	for _, net := range networks {
		a, err = gonet.ResolveUDPAddr(net, multicastAddr)
		if err == nil {
			network = net
			break
		}
	}
	if err != nil {
		return nil, err
	}

	ifaces, err := gonet.Interfaces()
	if err != nil {
		return nil, err
	}

	mcastListener, err := net.NewListenUDP(network, multicastAddr)
	if err != nil {
		return nil, err
	}

	var anySet bool
	for i := range ifaces {
		iface := ifaces[i]
		err = mcastListener.JoinGroup(&iface, a)
		if err == nil {
			anySet = true
		}
		if err != nil {
			log.Printf("cannot JoinGroup(%v, %v): %v", iface, a, err)
		}
	}
	if !anySet {
		_ = mcastListener.Close()
		return nil, fmt.Errorf("cannot JoinGroup(%v): %w", a, err)
	}

	err = mcastListener.SetMulticastLoopback(true)
	if err != nil {
		_ = mcastListener.Close()
		return nil, err
	}

	return mcastListener, nil
}

func newConn(network, port string) (*net.UDPConn, error) {
	return net.NewListenUDP(network, ":"+port)
}

func getLogContent(r *pool.Message) string {
	content := ""
	if r == nil {
		return content
	}
	body := r.Body()
	if body == nil {
		return content
	}
	defer func() {
		_, _ = body.Seek(0, io.SeekStart)
	}()
	contentFormat := message.TextPlain
	if m, err := r.Options().ContentFormat(); err == nil {
		contentFormat = m
	}

	switch contentFormat {
	case message.AppCBOR, message.AppOcfCbor:
		var v interface{}
		if err := cbor.ReadFrom(body, &v); err == nil {
			if data, err := json.Encode(v); err == nil {
				content = string(data)
			}
		}
	case message.TextPlain:
		data, err := io.ReadAll(body)
		if err == nil {
			content = string(data)
		}
	}
	return content
}

func logReqResp(c mux.Conn, r *mux.Message, resp *pool.Message) {
	content := getLogContent(resp)
	p, err := r.Path()
	if err == nil && p == "/.well-known/core" {
		// don't log core discovery
		return
	}
	respStr := ""
	if resp != nil {
		respStr = resp.String()
	}
	log.Printf("%v, req=%v resp=%v, content=%v\n", c.RemoteAddr(), r.String(), respStr, content)
}

func CreateResponseError(ctx context.Context, err error, token message.Token) *pool.Message {
	if err == nil {
		return nil
	}
	s, ok := status.FromError(err)
	code := codes.BadRequest
	if ok {
		code = s.Code()
	}
	msg := pool.NewMessage(ctx)
	msg.SetCode(code)
	msg.SetToken(token)
	// Don't set content format for diagnostic message: https://tools.ietf.org/html/rfc7252#section-5.5.2
	msg.SetBody(bytes.NewReader([]byte(err.Error())))
	return msg
}

func LoggingMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		next.ServeCOAP(w, r)
		logReqResp(w.Conn(), r, w.Message())
	})
}

func (n *Net) ServeCOAP(w mux.ResponseWriter, request *mux.Message) {
	now := time.Now()
	messageID := request.MessageID()
	if messageID >= 0 && request.Type() != message.Confirmable {
		v, loaded := n.cache.LoadOrStore(messageID, coapCache.NewElement(true, now.Add(n.cfg.DeduplicationLifetime), func(bool) {
		}))
		if loaded && !v.IsExpired(now) {
			log.Printf("duplicate message %v according messageID: %v", request, messageID)
			return
		}
	}
	request.Hijack()
	go func(w mux.ResponseWriter, request *mux.Message) {
		r := Request{
			Message:   request.Message,
			Endpoints: n.GetEndpoints(request.ControlMessage(), w.Conn().NetConn().LocalAddr().String()),
			Conn:      w.Conn(),
		}

		resp, err := n.handler(&r)
		if err != nil {
			resp = CreateResponseError(request.Context(), err, request.Token())
		}
		if resp != nil {
			resp.SetToken(request.Token())
			logReqResp(w.Conn(), request, resp)
			err = w.Conn().WriteMessage(resp)
			if err != nil {
				log.Printf("cannot write response: %v", err)
			}
		}
	}(w, request)
}

type coAPServer struct {
	s *server.Server
	l *net.UDPConn
}

type coAPServers []coAPServer

func (s coAPServers) Stop() {
	for _, cs := range s {
		cs.s.Stop()
	}
}

func (s coAPServers) Close() error {
	var errors *multierror.Error
	for _, cs := range s {
		err := cs.l.Close()
		if err != nil {
			errors = multierror.Append(errors, err)
		}
	}
	return errors.ErrorOrNil()
}

func getPortFromAddress(addr gonet.Addr) (string, error) {
	udpAddr, ok := addr.(*gonet.UDPAddr)
	if ok {
		return fmt.Sprintf("%d", udpAddr.Port), nil
	}
	addrStr := addr.String()
	_, port, err := gonet.SplitHostPort(addrStr)
	if err != nil {
		return "", err
	}
	return port, nil
}

func newServers(cfg *Config, m *mux.Router) (coAPServers, bool, bool, error) {
	servers := make(coAPServers, 0, len(cfg.externalAddressesPort))
	hasIPv4 := false
	hasIPv6 := false
	for i, addr := range cfg.externalAddressesPort {
		var conn *net.UDPConn
		var err error
		if addr.network == UDP4 {
			hasIPv4 = true
			conn, err = newConn(addr.network, addr.port)
		}
		if addr.network == UDP6 {
			hasIPv6 = true
			conn, err = newConn(addr.network, addr.port)
		}
		if err != nil {
			_ = servers.Close()
			return nil, false, false, err
		}
		if addr.port == "0" {
			port, err := getPortFromAddress(conn.LocalAddr())
			if err != nil {
				_ = servers.Close()
				return nil, false, false, err
			}
			cfg.externalAddressesPort[i].port = port
		}

		if conn != nil {
			servers = append(servers, coAPServer{
				s: udp.NewServer(
					options.WithMux(m),
					options.WithErrors(func(err error) { log.Printf("server: %v", err) }),
					options.WithMaxMessageSize(cfg.MaxMessageSize),
				),
				l: conn,
			})
		}
	}
	if len(servers) == 0 {
		return nil, false, false, fmt.Errorf("cannot create any server")
	}
	return servers, hasIPv4, hasIPv6, nil
}

func appendMCastServers(servers coAPServers, mcastAddresses []string, cfg Config, m *mux.Router) (coAPServers, error) {
	for _, addr := range mcastAddresses {
		if addr == "" {
			continue
		}
		conn, err := newMCastConn(addr)
		if err != nil {
			_ = servers.Close()
			return nil, err
		}
		servers = append(servers, coAPServer{
			s: udp.NewServer(options.WithMux(m),
				options.WithMaxMessageSize(cfg.MaxMessageSize),
			),
			l: conn,
		})
	}
	return servers, nil
}

func New(cfg Config, handler RequestHandler) (*Net, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	m := mux.NewRouter()
	servers, hasIPv4, hasIPv6, err := newServers(&cfg, m)
	if err != nil {
		return nil, err
	}
	if hasIPv4 {
		servers, err = appendMCastServers(servers, core.DefaultDiscoveryConfiguration().MulticastAddressUDP4, cfg, m)
		if err != nil {
			return nil, err
		}
	}
	if hasIPv6 {
		servers, err = appendMCastServers(servers, core.DefaultDiscoveryConfiguration().MulticastAddressUDP6, cfg, m)
		if err != nil {
			return nil, err
		}
	}

	n := &Net{
		cfg:     cfg,
		servers: servers,
		mux:     m,
		handler: handler,
		done:    make(chan struct{}),
		cache:   coapCache.NewCache[int32, bool](),
	}
	m.DefaultHandle(mux.HandlerFunc(n.ServeCOAP))
	go func() {
		for {
			select {
			case <-n.done:
				return
			case <-time.After(n.cfg.DeduplicationLifetime / 2):
				now := time.Now()
				n.cache.CheckExpirations(now)
			}
		}
	}()
	return n, nil
}

func (n *Net) GetEndpoints(cm *net.ControlMessage, localAddr string) schema.Endpoints {
	_, localPort, err := gonet.SplitHostPort(localAddr)
	if err != nil {
		log.Printf("cannot get local address: %v", err)
		return nil
	}
	network := UDP4
	if cm.Dst.To4() == nil {
		network = UDP6
	}
	filteredByNetwork := n.cfg.externalAddressesPort.filterByNetwork(network)
	filtered := filteredByNetwork.filterByPort(localPort)
	if len(filtered) == 0 {
		filtered = filteredByNetwork
	}
	ep := localAddr
	if len(filtered) > 0 {
		ep = filtered[0].host
		if filtered[0].network == UDP6 {
			ep = "[" + ep + "]"
		}
		ep = ep + ":" + filtered[0].port
	}
	return schema.Endpoints{
		{
			URI: fmt.Sprintf("coap://%v", ep),
		},
	}
}

func (n *Net) Serve() error {
	if !n.serving.CompareAndSwap(false, true) {
		return fmt.Errorf("already serving")
	}
	defer close(n.done)
	var wg sync.WaitGroup
	errCh := make(chan error, len(n.servers))
	wg.Add(len(n.servers))
	for _, cs := range n.servers {
		go func(cs coAPServer) {
			defer wg.Done()
			err := cs.s.Serve(cs.l)
			errCh <- err
		}(cs)
	}
	wg.Wait()
	var errors *multierror.Error
	for {
		select {
		case err := <-errCh:
			if err != nil {
				errors = multierror.Append(errors, err)
			}
		default:
			return errors.ErrorOrNil()
		}
	}
}

func (n *Net) Close() error {
	if !n.serving.Load() {
		return nil
	}
	n.servers.Stop()
	<-n.done
	return nil
}
