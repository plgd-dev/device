// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package coap

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	piondtls "github.com/pion/dtls/v2"
	codecOcf "github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/go-coap/v3/dtls"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/message/status"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/go-coap/v3/udp"
)

type Observation = interface {
	Cancel(context.Context, ...message.Option) error
	Canceled() bool
}

type ClientConn = interface {
	Post(ctx context.Context, path string, contentFormat message.MediaType, payload io.ReadSeeker, opts ...message.Option) (*pool.Message, error)
	Get(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error)
	Delete(ctx context.Context, path string, opts ...message.Option) (*pool.Message, error)
	Observe(ctx context.Context, path string, observeFunc func(notification *pool.Message), opts ...message.Option) (Observation, error)
	RemoteAddr() net.Addr
	Close() error
	Context() context.Context
	Done() <-chan struct{}
}

type Client struct {
	conn ClientConn
}

// Codec encodes/decodes according to the CoAP content format/media type.
type Codec interface {
	ContentFormat() message.MediaType
	Encode(v interface{}) ([]byte, error)
	Decode(m *pool.Message, v interface{}) error
}

func NewClient(conn ClientConn) *Client {
	return &Client{conn: conn}
}

type OptionFunc = func(message.Options) message.Options

func WithQuery(query string) OptionFunc {
	return func(opts message.Options) message.Options {
		buf := make([]byte, len(query))
		opts, _, _ = opts.AddString(buf, message.URIQuery, query)
		return opts
	}
}

func WithInterface(in string) OptionFunc {
	return WithQuery("if=" + in)
}

func WithResourceType(in string) OptionFunc {
	return WithQuery("rt=" + in)
}

func WithDeviceID(in string) OptionFunc {
	return WithQuery("di=" + in)
}

func WithAccept(contentFormat message.MediaType) OptionFunc {
	return func(opts message.Options) message.Options {
		buf := make([]byte, 4)
		opts, _, _ = opts.SetUint32(buf, message.Accept, uint32(contentFormat))
		return opts
	}
}

func WithETag(etag []byte) OptionFunc {
	return func(opts message.Options) message.Options {
		buf := make([]byte, len(etag))
		opts, _, _ = opts.SetBytes(buf, message.ETag, etag)
		return opts
	}
}

func (c *Client) UpdateResource(
	ctx context.Context,
	href string,
	request interface{},
	response interface{},
	options ...OptionFunc,
) error {
	return c.UpdateResourceWithCodec(ctx, href, codecOcf.VNDOCFCBORCodec{}, request, response, options...)
}

func requestError(m *pool.Message) error {
	return fmt.Errorf("request failed: %s", codecOcf.Dump(m))
}

func decodeError(href string, err error) error {
	return fmt.Errorf("could not decode the response %s: %w", href, err)
}

func (c *Client) UpdateResourceWithCodec(
	ctx context.Context,
	href string,
	codec Codec,
	request interface{},
	response interface{},
	options ...OptionFunc,
) error {
	body, err := codec.Encode(request)
	if err != nil {
		return fmt.Errorf("could not encode the request %s: %w", href, err)
	}
	opts := make(message.Options, 0, 4)
	for _, o := range options {
		opts = o(opts)
	}

	resp, err := c.conn.Post(ctx, href, codec.ContentFormat(), bytes.NewReader(body), opts...)
	if err != nil {
		return fmt.Errorf("could update request %s: %w", href, err)
	}
	response, err = TrySetDetailedReponse(resp, response)
	if err != nil {
		return err
	}
	if resp.Code() != codes.Changed && resp.Code() != codes.Valid && resp.Code() != codes.Created {
		return status.Error(resp, requestError(resp))
	}
	if resp.Code() == codes.Valid {
		return nil
	}
	if err := codec.Decode(resp, response); err != nil {
		return status.Error(resp, decodeError(href, err))
	}
	return nil
}

func (c *Client) GetResource(
	ctx context.Context,
	href string,
	response interface{},
	options ...OptionFunc,
) error {
	return c.GetResourceWithCodec(ctx, href, codecOcf.VNDOCFCBORCodec{}, response, options...)
}

func (c *Client) GetResourceWithCodec(
	ctx context.Context,
	href string,
	codec Codec,
	response interface{},
	options ...OptionFunc,
) error {
	opts := make(message.Options, 0, 4)
	for _, o := range options {
		opts = o(opts)
	}
	resp, err := c.conn.Get(ctx, href, opts...)
	if err != nil {
		return fmt.Errorf("could not get %s: %w", href, err)
	}

	response, err = TrySetDetailedReponse(resp, response)
	if err != nil {
		return err
	}

	switch code := resp.Code(); {
	case code == codes.Content:
		if errD := codec.Decode(resp, response); errD != nil {
			return status.Error(resp, decodeError(href, errD))
		}
	case code != codes.Valid:
		return status.Error(resp, requestError(resp))
	}
	return nil
}

func (c *Client) DeleteResourceWithCodec(
	ctx context.Context,
	href string,
	codec Codec,
	response interface{},
	options ...OptionFunc,
) error {
	opts := make(message.Options, 0, 4)
	for _, o := range options {
		opts = o(opts)
	}
	resp, err := c.conn.Delete(ctx, href, opts...)
	if err != nil {
		return fmt.Errorf("could not delete %s: %w", href, err)
	}
	if resp.Code() != codes.Deleted {
		return status.Error(resp, requestError(resp))
	}
	if err := codec.Decode(resp, response); err != nil {
		return status.Error(resp, decodeError(href, err))
	}
	return nil
}

func (c *Client) DeleteResource(
	ctx context.Context,
	href string,
	response interface{},
	options ...OptionFunc,
) error {
	return c.DeleteResourceWithCodec(ctx, href, codecOcf.VNDOCFCBORCodec{}, response, options...)
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// DecodeFunc can be used to pass in the data type that should be decoded.
type DecodeFunc = func(interface{}) error

// ObservationHandler receives notifications from the observation request.
type ObservationHandler interface {
	Handle(client *Client, body DecodeFunc)
	Error(err error)
	Close()
}

// Observe makes a CoAP observation request over a connection.
func (c *Client) Observe(
	ctx context.Context,
	href string,
	codec Codec,
	handler ObservationHandler,
	options ...OptionFunc,
) (Observation, error) {
	opts := make(message.Options, 0, 4)
	for _, o := range options {
		opts = o(opts)
	}
	obs, err := c.conn.Observe(ctx, href, observationHandler(c, codec, handler), opts...)
	if err != nil {
		return nil, fmt.Errorf("could not observe %s: %w", href, err)
	}
	return obs, nil
}

func observationHandler(c *Client, codec Codec, handler ObservationHandler) func(*pool.Message) {
	return func(msg *pool.Message) {
		_, err := msg.Options().Observe()
		// If msg doesn't contains observe option it means the resource doesn't support observation.
		doClose := err != nil
		handler.Handle(c, decodeObservation(codec, msg))
		if doClose {
			handler.Close()
		}
	}
}

func decodeObservation(codec Codec, m *pool.Message) DecodeFunc {
	return func(body interface{}) error {
		if m.Code() != codes.Content && m.Code() != codes.Valid {
			return status.Error(m, fmt.Errorf("observation failed: %s", codecOcf.Dump(m)))
		}
		if errD := codec.Decode(m, body); errD != nil {
			return status.Error(m, fmt.Errorf("could not decode observation: %w", errD))
		}
		return nil
	}
}

func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *Client) Context() context.Context {
	return c.conn.Context()
}

func (c *Client) Done() <-chan struct{} {
	return c.conn.Done()
}

type CloseHandlerFunc = func(err error)

type OnCloseHandler struct {
	handlers map[int]CloseHandlerFunc
	nextId   int
	lock     sync.Mutex
}

func NewOnCloseHandler() *OnCloseHandler {
	return &OnCloseHandler{
		handlers: make(map[int]CloseHandlerFunc),
	}
}

func (h *OnCloseHandler) Add(onClose func(err error)) int {
	h.lock.Lock()
	defer h.lock.Unlock()
	v := h.nextId
	h.nextId++
	h.handlers[v] = onClose
	return v
}

func (h *OnCloseHandler) Remove(onCloseID int) {
	h.lock.Lock()
	defer h.lock.Unlock()
	delete(h.handlers, onCloseID)
}

func (h *OnCloseHandler) getHandlers() []CloseHandlerFunc {
	h.lock.Lock()
	defer h.lock.Unlock()

	res := make([]func(error), 0, len(h.handlers))
	for _, ho := range h.handlers {
		res = append(res, ho)
	}
	return res
}

func (h *OnCloseHandler) OnClose(err error) {
	handlers := h.getHandlers()
	for _, ho := range handlers {
		ho(err)
	}
}

type ClientCloseHandler struct {
	*Client
	onClose *OnCloseHandler
}

func (c *ClientCloseHandler) RegisterCloseHandler(f CloseHandlerFunc) (closeHandlerID int) {
	return c.onClose.Add(f)
}

func (c *ClientCloseHandler) UnregisterCloseHandler(closeHandlerID int) {
	c.onClose.Remove(closeHandlerID)
}

func NewClientCloseHandler(conn ClientConn, onClose *OnCloseHandler) *ClientCloseHandler {
	return &ClientCloseHandler{Client: NewClient(conn), onClose: onClose}
}

func DialUDP(ctx context.Context, addr string, opts ...udp.Option) (*ClientCloseHandler, error) {
	h := NewOnCloseHandler()
	dopts := make([]udp.Option, 0, len(opts)+4)
	dopts = append(dopts, getDefaultUDPOptions(ctx)...)
	dopts = append(dopts, opts...)
	c, err := udp.Dial(addr, dopts...)
	if err != nil {
		return nil, err
	}
	c.AddOnClose(func() {
		h.OnClose(nil)
	})
	return NewClientCloseHandler(c, h), nil
}

func DialTCP(ctx context.Context, addr string, opts ...tcp.Option) (*ClientCloseHandler, error) {
	h := NewOnCloseHandler()
	dopts := make([]tcp.Option, 0, len(opts)+4)
	dopts = append(dopts, getDefaultTCPOptions(ctx, nil)...)
	dopts = append(dopts, opts...)
	c, err := tcp.Dial(addr, dopts...)
	if err != nil {
		return nil, err
	}
	c.AddOnClose(func() {
		h.OnClose(nil)
	})
	return NewClientCloseHandler(c, h), nil
}

func NewVerifyPeerCertificate(rootCAs *x509.CertPool, verifyPeerCertificate func(verifyPeerCertificate *x509.Certificate) error) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return fmt.Errorf("empty certificates chain")
		}
		intermediateCAPool := x509.NewCertPool()
		certs := make([]*x509.Certificate, 0, len(rawCerts))
		for _, rawCert := range rawCerts {
			cert, err := x509.ParseCertificate(rawCert)
			if err != nil {
				return err
			}
			certs = append(certs, cert)
		}
		for _, cert := range certs[1:] {
			intermediateCAPool.AddCert(cert)
		}
		_, err := certs[0].Verify(x509.VerifyOptions{
			Roots:         rootCAs,
			Intermediates: intermediateCAPool,
			CurrentTime:   time.Now(),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		})
		if err != nil {
			return err
		}
		if verifyPeerCertificate == nil {
			return nil
		}
		if verifyPeerCertificate(certs[0]) != nil {
			return err
		}
		return nil
	}
}

func getDefaultTCPOptions(ctx context.Context, tlsCfg *tls.Config) []tcp.Option {
	dopts := make([]tcp.Option, 0, 4)
	dopts = append(dopts, options.WithErrors(func(error) {
		// ignore by default
	}), options.WithMessagePool(pool.New(0, 0)))
	if tlsCfg != nil {
		dopts = append(dopts, options.WithTLS(tlsCfg))
	}
	deadline, ok := ctx.Deadline()
	if ok {
		dopts = append(dopts, options.WithDialer(&net.Dialer{
			Timeout: time.Until(deadline),
		}))
	}
	return dopts
}

func DialTCPSecure(ctx context.Context, addr string, tlsCfg *tls.Config, opts ...tcp.Option) (*ClientCloseHandler, error) {
	h := NewOnCloseHandler()
	dopts := make([]tcp.Option, 0, len(opts)+4)
	dopts = append(dopts, getDefaultTCPOptions(ctx, tlsCfg)...)
	dopts = append(dopts, opts...)
	c, err := tcp.Dial(addr, dopts...)
	if err != nil {
		return nil, err
	}
	c.AddOnClose(func() {
		h.OnClose(nil)
	})
	return NewClientCloseHandler(c, h), nil
}

func getDefaultUDPOptions(ctx context.Context) []udp.Option {
	dopts := make([]udp.Option, 0, 4)
	dopts = append(dopts, options.WithErrors(func(error) {
		// ignore by default
	}), options.WithMessagePool(pool.New(0, 0)))
	deadline, ok := ctx.Deadline()
	if ok {
		dopts = append(dopts, options.WithDialer(&net.Dialer{
			Timeout: time.Until(deadline),
		}))
	}
	return dopts
}

func DialUDPSecure(ctx context.Context, addr string, dtlsCfg *piondtls.Config, opts ...udp.Option) (*ClientCloseHandler, error) {
	h := NewOnCloseHandler()

	if dtlsCfg.ConnectContextMaker == nil {
		dtlsCfg.ConnectContextMaker = func() (context.Context, func()) {
			return ctx, func() {
				// no-op
			}
		}
	}
	dopts := make([]udp.Option, 0, len(opts)+4)
	dopts = append(dopts, getDefaultUDPOptions(ctx)...)
	dopts = append(dopts, opts...)
	c, err := dtls.Dial(addr, dtlsCfg, dopts...)
	if err != nil {
		return nil, err
	}
	c.AddOnClose(func() {
		h.OnClose(nil)
	})
	return NewClientCloseHandler(c, h), nil
}
