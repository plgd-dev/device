package local

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"sync"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/local/resource/link"
	"github.com/go-ocf/sdk/schema"
	"github.com/gofrs/uuid"
)

// Client an OCF local client.
type Client struct {
	observations *sync.Map
	factory      ResourceClientFactory
	conn         []*gocoap.MulticastClientConn

	tlsConfig resource.TLSConfig
}

// Config for the OCF local client.
type Config struct {
	Protocol  string
	Resource  resource.Config
	TLSConfig resource.TLSConfig
}

// NewClientFromConfig constructs a new OCF client.
func NewClientFromConfig(cfg Config, errors func(error)) (*Client, error) {
	conn := resource.DialDiscoveryAddresses(context.Background(), errors)
	linkCache, err := resource.NewLinkCache(cfg.Resource, conn)
	if err != nil {
		return nil, err
	}

	// Only TCP is supported at the moment.
	f := NewResourceClientFactory(cfg, linkCache)

	return NewClient(cfg.TLSConfig, f, conn), nil
}

func checkTLSConfig(cfg resource.TLSConfig) resource.TLSConfig {
	if cfg.GetCertificate == nil {
		cfg.GetCertificate = func() (tls.Certificate, error) {
			return tls.Certificate{}, fmt.Errorf("not supported")
		}
	}
	if cfg.GetCertificateAuthorities == nil {
		cfg.GetCertificateAuthorities = func() ([]*x509.Certificate, error) {
			return nil, fmt.Errorf("not supported")
		}
	}
	return cfg
}

func NewClient(TLSConfig resource.TLSConfig, f ResourceClientFactory, conn []*gocoap.MulticastClientConn) *Client {
	TLSConfig = checkTLSConfig(TLSConfig)
	return &Client{tlsConfig: TLSConfig, factory: f, conn: conn, observations: &sync.Map{}}
}

func NewResourceClientFactory(cfg Config, linkCache *link.Cache) ResourceClientFactory {
	cfg.TLSConfig = checkTLSConfig(cfg.TLSConfig)
	switch cfg.Protocol {
	case "tcp":
		return &tcpClientFactory{f: resource.NewTCPClientFactory(cfg.TLSConfig, linkCache)}
	case "udp":
		return &udpClientFactory{f: resource.NewUDPClientFactory(linkCache)}
	default:
		panic(fmt.Errorf("unsupported resource client protocol %s", cfg.Protocol))
	}
}

type resourceClient interface {
	Observe(ctx context.Context, deviceID, href string, codec resource.Codec, handler resource.ObservationHandler, options ...func(gocoap.Message)) (*gocoap.Observation, error)
	Get(ctx context.Context, deviceID, href string, codec resource.Codec, responseBody interface{}, options ...func(gocoap.Message)) error
	Post(ctx context.Context, deviceID, href string, codec resource.Codec, requestBody interface{}, responseBody interface{}, options ...func(gocoap.Message)) error
}

type ResourceClientFactory interface {
	NewClient(c *gocoap.ClientConn, links schema.DeviceLinks) (resourceClient, error)
	NewClientFromCache() (resourceClient, error)
	CloseConnections(links schema.DeviceLinks)
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

func (w *tcpClientFactory) CloseConnections(links schema.DeviceLinks) {
	w.f.CloseConnections(links)
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

func (w *udpClientFactory) CloseConnections(links schema.DeviceLinks) {
	w.f.CloseConnections(links)
}

func (c *Client) GetCertificate() (res tls.Certificate, _ error) {
	if c.tlsConfig.GetCertificate != nil {
		return c.tlsConfig.GetCertificate()
	}
	return res, fmt.Errorf("Config.GetCertificate is not set")
}

func (c *Client) GetCertificateAuthorities() (res []*x509.Certificate, _ error) {
	if c.tlsConfig.GetCertificateAuthorities != nil {
		return c.tlsConfig.GetCertificateAuthorities()
	}
	return res, fmt.Errorf("Config.GetCertificateAuthorities is not set")
}

func getDeviceIdFromCertificate(cert *x509.Certificate) (string, error) {
	// verify EKU manually
	ekuHasClient := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageClientAuth {
			ekuHasClient = true
			break
		}
	}
	if !ekuHasClient {
		return "", fmt.Errorf("not contains ExtKeyUsageClientAuth")
	}
	ekuHasOcfId := false
	for _, eku := range cert.UnknownExtKeyUsage {
		if eku.Equal(schema.ExtendedKeyUsage_IDENTITY_CERTIFICATE) {
			ekuHasOcfId = true
			break
		}
	}
	if !ekuHasOcfId {
		return "", fmt.Errorf("not contains ExtKeyUsage with OCF ID(1.3.6.1.4.1.44924.1.6")
	}
	cn := strings.Split(cert.Subject.CommonName, ":")
	if len(cn) != 2 {
		return "", fmt.Errorf("invalid subject common name: %v", cert.Subject.CommonName)
	}
	if strings.ToLower(cn[0]) != "uuid" {
		return "", fmt.Errorf("invalid subject common name %v: 'uuid' - not found", cert.Subject.CommonName)
	}
	deviceId, err := uuid.FromString(cn[1])
	if err != nil {
		return "", fmt.Errorf("invalid subject common name %v: %v", cert.Subject.CommonName, err)
	}
	return deviceId.String(), nil
}

// GetSdkDeviceID returns sdk deviceID from identity certificate.
func (c *Client) GetSdkDeviceID() (string, error) {
	cert, err := c.GetCertificate()
	if err != nil {
		return "", fmt.Errorf("cannot get sdk id: %v", err)
	}

	var errors []error

	for _, c := range cert.Certificate {
		x509cert, err := x509.ParseCertificate(c)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		deviceId, err := getDeviceIdFromCertificate(x509cert)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		return deviceId, nil
	}
	return "", fmt.Errorf("cannot get sdk id: %v", errors)
}

func (c *Client) CloseConnections(links schema.DeviceLinks) {
	c.factory.CloseConnections(links)
}
