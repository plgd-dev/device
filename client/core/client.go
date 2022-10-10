package core

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"

	"github.com/pion/dtls/v2"
	"github.com/pion/logging"
	pkgError "github.com/plgd-dev/device/v2/pkg/error"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	coapNet "github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/go-coap/v3/udp"
)

// Client an OCF local client.
type Client struct {
	tlsConfig *TLSConfig
	logger    Logger
	dialDTLS  DialDTLS
	dialTLS   DialTLS
	dialTCP   DialTCP
	dialUDP   DialUDP
}

func checkTLSConfig(cfg *TLSConfig) *TLSConfig {
	if cfg == nil {
		cfg = new(TLSConfig)
	}
	if cfg.GetCertificate == nil {
		cfg.GetCertificate = func() (tls.Certificate, error) {
			return tls.Certificate{}, MakeUnimplemented(pkgError.NotSupported())
		}
	}
	if cfg.GetCertificateAuthorities == nil {
		cfg.GetCertificateAuthorities = func() ([]*x509.Certificate, error) {
			return nil, MakeUnimplemented(pkgError.NotSupported())
		}
	}
	return cfg
}

type config struct {
	tlsConfig *TLSConfig
	logger    Logger
	dialDTLS  DialDTLS
	dialTLS   DialTLS
	dialTCP   DialTCP
	dialUDP   DialUDP
}

type OptionFunc func(config) config

func WithTLS(tlsConfig *TLSConfig) OptionFunc {
	return func(cfg config) config {
		if tlsConfig != nil {
			cfg.tlsConfig = tlsConfig
		}
		return cfg
	}
}

// DiscoveryConfiguration setup discovery configuration
type DiscoveryConfiguration struct {
	MulticastHopLimit    int      // default: 2, min value: 1 - don't pass through router, max value: 255, https://tools.ietf.org/html/rfc2460#section-3
	MulticastAddressUDP4 []string // default: "[224.0.1.187:5683] (client.DiscoveryAddressUDP4), empty: don't use ipv4 multicast"
	MulticastAddressUDP6 []string // default: "[ff02::158]:5683", "[ff03::158]:5683", "[ff05::158]:5683]"] (client.DiscoveryAddressUDP6), empty: don't use ipv6 multicast"
	MulticastOptions     []coapNet.MulticastOption
}

func WithLogger(logger Logger) OptionFunc {
	return func(cfg config) config {
		if logger != nil {
			cfg.logger = logger
		}
		return cfg
	}
}

type (
	DialDTLS = func(ctx context.Context, addr string, dtlsCfg *dtls.Config, opts ...udp.Option) (*coap.ClientCloseHandler, error)
	DialTLS  = func(ctx context.Context, addr string, tlsCfg *tls.Config, opts ...tcp.Option) (*coap.ClientCloseHandler, error)
	DialUDP  = func(ctx context.Context, addr string, opts ...udp.Option) (*coap.ClientCloseHandler, error)
	DialTCP  = func(ctx context.Context, addr string, opts ...tcp.Option) (*coap.ClientCloseHandler, error)
)

func WithDialDTLS(dial DialDTLS) OptionFunc {
	return func(cfg config) config {
		if dial != nil {
			cfg.dialDTLS = dial
		}
		return cfg
	}
}

func WithDialTLS(dial DialTLS) OptionFunc {
	return func(cfg config) config {
		if dial != nil {
			cfg.dialTLS = dial
		}
		return cfg
	}
}

func WithDialTCP(dial DialTCP) OptionFunc {
	return func(cfg config) config {
		if dial != nil {
			cfg.dialTCP = dial
		}
		return cfg
	}
}

func WithDialUDP(dial DialUDP) OptionFunc {
	return func(cfg config) config {
		if dial != nil {
			cfg.dialUDP = dial
		}
		return cfg
	}
}

func (c *Client) getDeviceConfiguration() DeviceConfiguration {
	return DeviceConfiguration{
		Logger:    c.logger,
		DialDTLS:  c.dialDTLS,
		DialTLS:   c.dialTLS,
		DialTCP:   c.dialTCP,
		DialUDP:   c.dialUDP,
		TLSConfig: c.tlsConfig,
	}
}

func DefaultDiscoveryConfiguration() DiscoveryConfiguration {
	return DiscoveryConfiguration{
		MulticastHopLimit:    2,
		MulticastAddressUDP4: DiscoveryAddressUDP4,
		MulticastAddressUDP6: DiscoveryAddressUDP6,
		MulticastOptions: []coapNet.MulticastOption{coapNet.WithMulticastInterfaceError(func(iface *net.Interface, err error) {
			// ignore error
		})},
	}
}

func NewClient(opts ...OptionFunc) *Client {
	cfg := config{
		logger:   logging.NewDefaultLoggerFactory().NewLogger("client"),
		dialDTLS: coap.DialUDPSecure,
		dialTLS:  coap.DialTCPSecure,
		dialTCP:  coap.DialTCP,
		dialUDP:  coap.DialUDP,
	}
	for _, o := range opts {
		cfg = o(cfg)
	}

	cfg.tlsConfig = checkTLSConfig(cfg.tlsConfig)
	return &Client{
		dialDTLS:  cfg.dialDTLS,
		dialTLS:   cfg.dialTLS,
		dialTCP:   cfg.dialTCP,
		dialUDP:   cfg.dialUDP,
		logger:    cfg.logger,
		tlsConfig: cfg.tlsConfig,
	}
}
