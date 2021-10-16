package manufacturer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/pion/dtls/v2"
	"github.com/plgd-dev/device/client/core/otm"
	kitNetCoap "github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/doxm"
	kitNet "github.com/plgd-dev/kit/v2/net"
)

type DialDTLS = func(ctx context.Context, addr string, dtlsCfg *dtls.Config, opts ...kitNetCoap.DialOptionFunc) (*kitNetCoap.ClientCloseHandler, error)
type DialTLS = func(ctx context.Context, addr string, tlsCfg *tls.Config, opts ...kitNetCoap.DialOptionFunc) (*kitNetCoap.ClientCloseHandler, error)

type Client struct {
	manufacturerCertificate tls.Certificate
	manufacturerCA          []*x509.Certificate
	dialDTLS                DialDTLS
	dialTLS                 DialTLS

	otm.Signer
}

type OptionFunc func(Client) Client

func WithDialDTLS(dial DialDTLS) OptionFunc {
	return func(cfg Client) Client {
		if dial != nil {
			cfg.dialDTLS = dial
		}
		return cfg
	}
}

func WithDialTLS(dial DialTLS) OptionFunc {
	return func(cfg Client) Client {
		if dial != nil {
			cfg.dialTLS = dial
		}
		return cfg
	}
}

func NewClient(
	manufacturerCertificate tls.Certificate,
	manufacturerCA []*x509.Certificate,
	sign otm.SignFunc,
	opts ...OptionFunc,
) *Client {
	c := Client{
		manufacturerCertificate: manufacturerCertificate,
		manufacturerCA:          manufacturerCA,
		dialDTLS:                kitNetCoap.DialUDPSecure,
		dialTLS:                 kitNetCoap.DialTCPSecure,
		Signer: otm.Signer{
			Sign: sign,
		},
	}
	for _, o := range opts {
		c = o(c)
	}
	return &c
}

func (*Client) Type() doxm.OwnerTransferMethod {
	return doxm.ManufacturerCertificate
}

func (c *Client) Dial(ctx context.Context, addr kitNet.Addr, opts ...kitNetCoap.DialOptionFunc) (*kitNetCoap.ClientCloseHandler, error) {
	switch schema.Scheme(addr.GetScheme()) {
	case schema.UDPSecureScheme:
		rootCAs := x509.NewCertPool()
		for _, ca := range c.manufacturerCA {
			rootCAs.AddCert(ca)
		}

		tlsConfig := dtls.Config{
			InsecureSkipVerify:    true,
			CipherSuites:          []dtls.CipherSuiteID{dtls.TLS_ECDHE_ECDSA_WITH_AES_128_CCM_8, dtls.TLS_ECDHE_ECDSA_WITH_AES_128_CCM},
			Certificates:          []tls.Certificate{c.manufacturerCertificate},
			VerifyPeerCertificate: kitNetCoap.NewVerifyPeerCertificate(rootCAs, func(*x509.Certificate) error { return nil }),
		}
		return c.dialDTLS(ctx, addr.String(), &tlsConfig, opts...)
	case schema.TCPSecureScheme:
		rootCAs := x509.NewCertPool()
		for _, ca := range c.manufacturerCA {
			rootCAs.AddCert(ca)
		}
		tlsConfig := tls.Config{
			InsecureSkipVerify:    true,
			Certificates:          []tls.Certificate{c.manufacturerCertificate},
			VerifyPeerCertificate: kitNetCoap.NewVerifyPeerCertificate(rootCAs, func(*x509.Certificate) error { return nil }),
		}
		return c.dialTLS(ctx, addr.String(), &tlsConfig, opts...)
	}
	return nil, fmt.Errorf("cannot dial to url %v: scheme %v not supported", addr.URL(), addr.GetScheme())
}
