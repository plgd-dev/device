package local

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/local/resource/link"
	"github.com/go-ocf/sdk/schema"
)

// Client an OCF local client.
type Client struct {
	observations  *sync.Map
	Certificate   tls.Certificate
	CertificateId string
	ca            []x509.Certificate
	factory       ResourceClientFactory
	conn          []*gocoap.MulticastClientConn

	tlsConfig TLSConfig

	lock sync.Mutex
}

// GetCertificateFunc returns certificate for connection
type GetCertificateFunc func() (tls.Certificate, error)

// GetCertificateAuthoritiesFunc returns certificate authorities to verify peers
type GetCertificateAuthoritiesFunc func() ([]*x509.Certificate, error)

type TLSConfig struct {
	// Used by own device
	GetManufacurerCertificate             GetCertificateFunc
	GetManufacturerCertificateAuthorities GetCertificateAuthoritiesFunc
	// User for communication with owned devices and cloud
	GetCertificate            GetCertificateFunc
	GetCertificateAuthorities GetCertificateAuthoritiesFunc
}

// Config for the OCF local client.
type Config struct {
	Protocol  string
	Resource  resource.Config
	TLSConfig TLSConfig
}

// NewClientFromConfig constructs a new OCF client.
func NewClientFromConfig(cfg Config, errors func(error)) (*Client, error) {
	conn := resource.DialDiscoveryAddresses(context.Background(), errors)
	linkCache, err := resource.NewLinkCache(cfg.Resource, conn)
	if err != nil {
		return nil, err
	}

	// Only TCP is supported at the moment.
	f := NewResourceClientFactory(cfg.Protocol, linkCache)

	return NewClient(cfg.TLSConfig, f, conn), nil
}

func NewClient(TLSConfig TLSConfig, f ResourceClientFactory, conn []*gocoap.MulticastClientConn) *Client {
	return &Client{tlsConfig: TLSConfig, factory: f, conn: conn, observations: &sync.Map{}}
}

func NewResourceClientFactory(protocol string, linkCache *link.Cache) ResourceClientFactory {
	switch protocol {
	case "tcp":
		return &tcpClientFactory{f: resource.NewTCPClientFactory(linkCache)}
	case "udp":
		return &udpClientFactory{f: resource.NewUDPClientFactory(linkCache)}
	default:
		panic(fmt.Errorf("unsupported resource client protocol %s", protocol))
	}
}

type resourceClient interface {
	Observe(ctx context.Context, deviceID, href string, handler resource.ObservationHandler, options ...func(gocoap.Message)) (*gocoap.Observation, error)
	Get(ctx context.Context, deviceID, href string, codec resource.Codec, responseBody interface{}, options ...func(gocoap.Message)) error
	Post(ctx context.Context, deviceID, href string, codec resource.Codec, requestBody interface{}, responseBody interface{}, options ...func(gocoap.Message)) error
}

type ResourceClientFactory interface {
	NewClient(c *gocoap.ClientConn, links schema.DeviceLinks) (resourceClient, error)
	NewClientFromCache() (resourceClient, error)
}

// tcpClientFactory converts the return type from *TCPClient to resourceClient.
type tcpClientFactory struct {
	f *resource.TCPClientFactory
}

func (w *tcpClientFactory) NewClient(c *gocoap.ClientConn, links schema.DeviceLinks) (resourceClient, error) {
	return w.f.NewClient(c, links)
}

func (w *tcpClientFactory) NewClientFromCache() (resourceClient, error) {
	return w.f.NewClientFromCache()
}

// udpClientFactory converts the return type from *UDPClient to resourceClient.
type udpClientFactory struct {
	f *resource.UDPClientFactory
}

func (w *udpClientFactory) NewClient(c *gocoap.ClientConn, links schema.DeviceLinks) (resourceClient, error) {
	return w.f.NewClient(c, links)
}

func (w *udpClientFactory) NewClientFromCache() (resourceClient, error) {
	return w.f.NewClientFromCache()
}

func (c *Client) GetManufacurerCertificate() (res tls.Certificate, _ error) {
	if c.tlsConfig.GetManufacurerCertificate != nil {
		return c.tlsConfig.GetManufacurerCertificate()
	}
	return res, fmt.Errorf("Config.GetManufacurerCertificate is not set")
}

func (c *Client) GetManufacturerCertificateAuthorities() (res []*x509.Certificate, _ error) {
	if c.tlsConfig.GetManufacturerCertificateAuthorities != nil {
		return c.tlsConfig.GetManufacturerCertificateAuthorities()
	}
	return res, fmt.Errorf("Config.GetManufacturerCertificateAuthorities is not set")
}

func (c *Client) GetCertificate() (res tls.Certificate, _ error) {
	if c.tlsConfig.GetCertificate != nil {
		return c.tlsConfig.GetCertificate()
	}
	return res, fmt.Errorf("Config.GetCertificate is not set")
}

func (c *Client) GetCertificateAuthorities() (res []*x509.Certificate, _ error) {
	if c.tlsConfig.GetManufacturerCertificateAuthorities != nil {
		return c.tlsConfig.GetManufacturerCertificateAuthorities()
	}
	return res, fmt.Errorf("Config.GetCertificateAuthorities is not set")
}
