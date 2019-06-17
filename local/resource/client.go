package resource

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/net"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
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
	getAddr   GetAddr
}

type GetAddr = func(schema.ResourceLink) (net.Addr, error)

// Get makes a GET CoAP request over a connection from the client's pool.
func (c *Client) Get(
	ctx context.Context,
	deviceID, href string,
	codec kitNetCoap.Codec,
	responseBody interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	conn, err := c.getConn(ctx, deviceID, href)
	if err != nil {
		return err
	}
	return kitNetCoap.NewClient(conn).GetResourceWithCodec(ctx, href, codec, responseBody, options...)
}

// Observe makes a CoAP observation request over a connection from the client's pool.
// It stores the observation context and returns an id.
func (c *Client) Observe(
	ctx context.Context,
	deviceID, href string,
	codec kitNetCoap.Codec,
	handler kitNetCoap.ObservationHandler,
	options ...kitNetCoap.OptionFunc,
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

	return kitNetCoap.NewClient(conn).Observe(ctx, href, codec, handler, options...)
}

// Post makes a POST CoAP request over a connection from the client's pool.
func (c *Client) Post(
	ctx context.Context,
	deviceID, href string,
	codec kitNetCoap.Codec,
	requestBody interface{},
	responseBody interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	conn, err := c.getConn(ctx, deviceID, href)
	if err != nil {
		return err
	}
	return kitNetCoap.NewClient(conn).UpdateResourceWithCodec(ctx, href, codec, requestBody, responseBody, options...)
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
	codec kitNetCoap.Codec,
	responseBody interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	return kitNetCoap.NewClient(conn).DeleteResourceWithCodec(ctx, href, codec, responseBody, options...)
}
