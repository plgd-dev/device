package justworks

import (
	"context"
	"fmt"

	"github.com/pion/dtls/v2"
	"github.com/plgd-dev/device/client/core/otm/just-works/cipher"
	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/doxm"
	"github.com/plgd-dev/go-coap/v3/udp"
	kitNet "github.com/plgd-dev/kit/v2/net"
)

type Client struct {
	dialDTLS DialDTLS
}

type DialDTLS = func(ctx context.Context, addr string, dtlsCfg *dtls.Config, opts ...udp.Option) (*coap.ClientCloseHandler, error)

type OptionFunc func(Client) Client

func WithDialDTLS(dial DialDTLS) OptionFunc {
	return func(cfg Client) Client {
		if dial != nil {
			cfg.dialDTLS = dial
		}
		return cfg
	}
}

func NewClient(opts ...OptionFunc) *Client {
	c := Client{
		dialDTLS: coap.DialUDPSecure,
	}
	for _, o := range opts {
		c = o(c)
	}
	return &c
}

func (*Client) Type() doxm.OwnerTransferMethod {
	return doxm.JustWorks
}

func (c *Client) Dial(ctx context.Context, addr kitNet.Addr) (*coap.ClientCloseHandler, error) {
	switch schema.Scheme(addr.GetScheme()) {
	case schema.UDPSecureScheme:
		tlsConfig := dtls.Config{
			CustomCipherSuites: func() []dtls.CipherSuite {
				return []dtls.CipherSuite{cipher.NewTLSAecdhAes128Sha256(dtls.CipherSuiteID(0xff00))}
			},
			CipherSuites: []dtls.CipherSuiteID{},
			ConnectContextMaker: func() (context.Context, func()) {
				return context.WithCancel(ctx)
			},
		}
		return c.dialDTLS(ctx, addr.String(), &tlsConfig)
	}
	return nil, fmt.Errorf("cannot dial to url %v: scheme %v not supported", addr.URL(), addr.GetScheme())
}
