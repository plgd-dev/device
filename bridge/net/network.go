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
	"math"
	gonet "net"
	"strconv"
	"strings"

	"github.com/google/uuid"
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
	"github.com/plgd-dev/go-coap/v3/udp"
	"github.com/plgd-dev/go-coap/v3/udp/server"
)

type Config struct {
	ExternalAddress     string `yaml:"externalAddress"`
	MaxMessageSize      uint32 `yaml:"maxMessageSize"`
	externalAddressPort string `yaml:"-"`
}

type RequestHandler func(req *Request) (*pool.Message, error)

type Net struct {
	cfg           Config
	listener      *net.UDPConn
	server        *server.Server
	mcastListener *net.UDPConn
	mcastServer   *server.Server
	handler       RequestHandler

	mux *mux.Router
}

func (cfg *Config) Validate() error {
	if cfg.ExternalAddress == "" {
		return fmt.Errorf("externalAddress is required")
	}
	host, portStr, err := gonet.SplitHostPort(cfg.ExternalAddress)
	if err != nil {
		return fmt.Errorf("invalid externalAddress: %w", err)
	}
	if host == "" {
		return fmt.Errorf("invalid externalAddress: host cannot be empty")
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid externalAddress: %w", err)
	}
	if port == 0 {
		return fmt.Errorf("invalid externalAddress: port cannot be 0")
	}
	if port > math.MaxUint16 {
		return fmt.Errorf("invalid externalAddress: port cannot be greater than %v", math.MaxUint16)
	}
	if cfg.MaxMessageSize == 0 {
		cfg.MaxMessageSize = 2 * 1024 * 1024
	}

	cfg.externalAddressPort = portStr
	return nil
}

// TODO: ipv6 + ipv6 multicast addresses
func initConnectivity(listenAddress string) (*net.UDPConn, *net.UDPConn, error) {
	multicastAddr := "224.0.1.187:5683"

	mcastListener, err := net.NewListenUDP("udp4", multicastAddr)
	if err != nil {
		return nil, nil, err
	}

	ifaces, err := gonet.Interfaces()
	if err != nil {
		_ = mcastListener.Close()
		return nil, nil, err
	}

	a, err := gonet.ResolveUDPAddr("udp4", multicastAddr)
	if err != nil {
		_ = mcastListener.Close()
		return nil, nil, err
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
		return nil, nil, fmt.Errorf("cannot JoinGroup(%v): %v", a, err)
	}

	err = mcastListener.SetMulticastLoopback(true)
	if err != nil {
		_ = mcastListener.Close()
		return nil, nil, err
	}

	l, err := net.NewListenUDP("udp4", listenAddress)
	if err != nil {
		_ = mcastListener.Close()
		return nil, nil, err
	}
	return mcastListener, l, nil
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

type Request struct {
	*pool.Message
	Conn      mux.Conn
	Endpoints schema.Endpoints
}

func (r *Request) Interface() string {
	q, err := r.Queries()
	if err != nil {
		return ""
	}
	for _, query := range q {
		if strings.HasPrefix(query, "if=") {
			return strings.TrimPrefix(query, "if=")
		}
	}
	return ""
}

func (r *Request) URIPath() string {
	p, err := r.Message.Options().Path()
	if err != nil {
		return ""
	}
	return p
}

func (r *Request) DeviceID() uuid.UUID {
	q, err := r.Queries()
	if err != nil {
		return uuid.Nil
	}
	for _, query := range q {
		if strings.HasPrefix(query, "di=") {
			deviceID := strings.TrimPrefix(query, "di=")
			di, err := uuid.Parse(deviceID)
			if err != nil {
				return uuid.Nil
			}
			return di
		}
	}
	return uuid.Nil
}

func (r *Request) ResourceTypes() []string {
	q, err := r.Queries()
	if err != nil {
		return nil
	}
	resourceTypes := make([]string, 0, len(q))
	for _, query := range q {
		if strings.HasPrefix(query, "rt=") {
			resourceTypes = append(resourceTypes, strings.TrimPrefix(query, "rt="))
		}
	}
	return resourceTypes
}

func (n *Net) ServeCOAP(w mux.ResponseWriter, request *mux.Message) {
	request.Hijack()
	go func(w mux.ResponseWriter, request *mux.Message) {
		r := Request{
			Message:   request.Message,
			Endpoints: n.GetEndpoints(),
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

func New(cfg Config, handler RequestHandler) (*Net, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	m := mux.NewRouter()
	mcastListener, listener, err := initConnectivity(fmt.Sprintf("0.0.0.0:%v", cfg.externalAddressPort))
	if err != nil {
		return nil, err
	}
	n := &Net{
		cfg:           cfg,
		listener:      listener,
		mcastListener: mcastListener,
		server: udp.NewServer(
			options.WithMux(m),
			options.WithErrors(func(err error) { log.Printf("server: %v", err) }),
			options.WithMaxMessageSize(cfg.MaxMessageSize),
		),
		mcastServer: udp.NewServer(options.WithMux(m),
			options.WithMaxMessageSize(cfg.MaxMessageSize),
		),
		mux:     m,
		handler: handler,
	}
	m.DefaultHandle(mux.HandlerFunc(n.ServeCOAP))
	return n, nil
}

func (n *Net) GetEndpoints() schema.Endpoints {
	return schema.Endpoints{
		{
			URI: fmt.Sprintf("coap://%v", n.cfg.ExternalAddress),
		},
	}
}

func (n *Net) Serve() error {
	go func() {
		err := n.mcastServer.Serve(n.mcastListener)
		if err != nil {
			log.Printf("mcastServer.Serve: %v", err)
		}
	}()
	return n.server.Serve(n.listener)
}

func (n *Net) Close() error {
	n.server.Stop()
	n.mcastServer.Stop()
	_ = n.mcastListener.Close()
	return n.listener.Close()
}
