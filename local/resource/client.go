package resource

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/kit/net"
	"github.com/go-ocf/kit/sync"
	"github.com/go-ocf/sdk/local/resource/link"
	"github.com/go-ocf/sdk/schema"
)

// GetCertificateFunc returns certificate for connection
type GetCertificateFunc func() (tls.Certificate, error)

// GetCertificateAuthoritiesFunc returns certificate authorities to verify peers
type GetCertificateAuthoritiesFunc func() ([]*x509.Certificate, error)

type TLSConfig struct {
	// User for communication with owned devices and cloud
	GetCertificate            GetCertificateFunc
	GetCertificateAuthorities GetCertificateAuthoritiesFunc
}

// Client caches resource links and maintains a pool of connections to devices.
type Client struct {
	linkCache *link.Cache
	pool      *sync.Pool
	//codec     Codec
	getAddr GetAddr
}

type GetAddr = func(schema.ResourceLink) (net.Addr, error)

// Codec encodes/decodes according to the CoAP content format/media type.
type Codec interface {
	ContentFormat() gocoap.MediaType
	Encode(v interface{}) ([]byte, error)
	Decode(m gocoap.Message, v interface{}) error
}

func COAPGet(
	ctx context.Context,
	conn *gocoap.ClientConn,
	href string,
	codec Codec,
	responseBody interface{},
	options ...func(gocoap.Message),
) error {
	req, err := conn.NewGetRequest(href)
	if err != nil {
		return fmt.Errorf("could create request %s: %v", href, err)
	}
	for _, option := range options {
		option(req)
	}
	resp, err := conn.ExchangeWithContext(ctx, req)
	if err != nil {
		return fmt.Errorf("could not query %s: %v", href, err)
	}
	if resp.Code() != gocoap.Content {
		return fmt.Errorf("request failed: %s", coap.Dump(resp))
	}
	if err := codec.Decode(resp, responseBody); err != nil {
		return fmt.Errorf("could not decode the query %s: %v", href, err)
	}
	return nil
}

// Get makes a GET CoAP request over a connection from the client's pool.
func (c *Client) Get(
	ctx context.Context,
	deviceID, href string,
	codec Codec,
	responseBody interface{},
	options ...func(gocoap.Message),
) error {
	conn, err := c.getConn(ctx, deviceID, href)
	if err != nil {
		return err
	}
	return COAPGet(ctx, conn, href, codec, responseBody, options...)
}

func COAPPost(
	ctx context.Context,
	conn *gocoap.ClientConn,
	href string,
	codec Codec,
	requestBody interface{},
	responseBody interface{},
	options ...func(gocoap.Message),
) error {
	body, err := codec.Encode(requestBody)
	if err != nil {
		return fmt.Errorf("could not encode the query %s: %v", href, err)
	}
	req, err := conn.NewPostRequest(href, codec.ContentFormat(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("could create request %s: %v", href, err)
	}
	for _, option := range options {
		option(req)
	}
	resp, err := conn.ExchangeWithContext(ctx, req)
	if err != nil {
		return fmt.Errorf("could not query %s: %v", href, err)
	}
	if resp.Code() != gocoap.Changed && resp.Code() != gocoap.Valid {
		return fmt.Errorf("request failed: %s", coap.Dump(resp))
	}
	if err := codec.Decode(resp, responseBody); err != nil {
		return fmt.Errorf("could not decode the query %s: %v", href, err)
	}
	return nil
}

// DecodeFunc can be used to pass in the data type that should be decoded.
type DecodeFunc func(interface{}) error

// ObservationHandler receives notifications from the observation request.
type ObservationHandler interface {
	Handle(ctx context.Context, client *gocoap.ClientConn, body DecodeFunc)
	Error(err error)
}

// Observe makes a CoAP observation request over a connection from the client's pool.
// It stores the observation context and returns an id.
func (c *Client) Observe(
	ctx context.Context,
	deviceID, href string,
	codec Codec,
	handler ObservationHandler,
	options ...func(gocoap.Message),
) (*gocoap.Observation, error) {
	r, err := c.linkCache.GetOrCreate(ctx, deviceID, href)
	if err != nil {
		return nil, fmt.Errorf("no response from device %s: %v", deviceID, err)
	}
	if !r.Policy.BitMask.Has(schema.Observable) {
		return nil, fmt.Errorf("non-observable resource %s", href)
	}
	conn, err := c.getConn(ctx, deviceID, href)
	if err != nil {
		return nil, err
	}
	obs, err := conn.ObserveWithContext(ctx, href, observationHandler(codec, handler), options...)
	if err != nil {
		return nil, fmt.Errorf("could not observe %s: %v", href, err)
	}
	return obs, nil
}

func observationHandler(codec Codec, handler ObservationHandler) func(*gocoap.Request) {
	return func(req *gocoap.Request) {
		handler.Handle(req.Ctx, req.Client, decodeObservation(codec, req.Msg))
	}
}

func decodeObservation(codec Codec, m gocoap.Message) DecodeFunc {
	return func(body interface{}) error {
		if m.Code() != gocoap.Content {
			return fmt.Errorf("observation failed: %s", coap.Dump(m))
		}
		if err := codec.Decode(m, body); err != nil {
			return fmt.Errorf("could not decode observation: %v", err)
		}
		return nil
	}
}

// Post makes a POST CoAP request over a connection from the client's pool.
func (c *Client) Post(
	ctx context.Context,
	deviceID, href string,
	codec Codec,
	requestBody interface{},
	responseBody interface{},
	options ...func(gocoap.Message),
) error {
	conn, err := c.getConn(ctx, deviceID, href)
	if err != nil {
		return err
	}
	return COAPPost(ctx, conn, href, codec, requestBody, responseBody, options...)
}

func (c *Client) getConn(ctx context.Context, deviceID, href string) (*gocoap.ClientConn, error) {
	r, err := c.linkCache.GetOrCreate(ctx, deviceID, href)
	if err != nil {
		return nil, fmt.Errorf("no response from device %s: %v", deviceID, err)
	}
	addr, err := c.getAddr(r)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint of device %s: %v", deviceID, err)
	}
	conn, err := c.pool.GetOrCreate(ctx, addr.URL())
	if err != nil {
		return nil, fmt.Errorf("could not connect to %s: %v", addr.String(), err)
	}
	return conn.(*gocoap.ClientConn), nil
}

func COAPDelete(
	ctx context.Context,
	conn *gocoap.ClientConn,
	href string,
	codec Codec,
	responseBody interface{},
	options ...func(gocoap.Message),
) error {
	req, err := conn.NewDeleteRequest(href)
	if err != nil {
		return fmt.Errorf("could create request %s: %v", href, err)
	}
	for _, option := range options {
		option(req)
	}
	resp, err := conn.ExchangeWithContext(ctx, req)
	if err != nil {
		return fmt.Errorf("could not query %s: %v", href, err)
	}
	if resp.Code() != gocoap.Deleted {
		return fmt.Errorf("request failed: %s", coap.Dump(resp))
	}
	if err := codec.Decode(resp, responseBody); err != nil {
		return fmt.Errorf("could not decode the query %s: %v", href, err)
	}
	return nil
}
