package local

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
)

// Client an OCF local client.
type Client struct {
	conn []*gocoap.MulticastClientConn

	tlsConfig *TLSConfig
}

// Config for the OCF local client.
type Config struct {
	TLSConfig *TLSConfig
}

// NewClientFromConfig constructs a new OCF client.
func NewClientFromConfig(cfg Config, errors func(error)) (*Client, error) {
	conn := DialDiscoveryAddresses(context.Background(), errors)

	return NewClient(cfg.TLSConfig, conn), nil
}

func checkTLSConfig(cfg *TLSConfig) *TLSConfig {
	if cfg == nil {
		cfg = new(TLSConfig)
	}
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

func NewClient(tlsConfig *TLSConfig, conn []*gocoap.MulticastClientConn) *Client {
	tlsConfig = checkTLSConfig(tlsConfig)
	return &Client{tlsConfig: tlsConfig, conn: conn}
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


