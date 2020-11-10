package core

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/plgd-dev/kit/net"
	"github.com/plgd-dev/kit/net/coap"

	"github.com/plgd-dev/kit/log"
)

// ErrFunc to log errors in goroutines
type ErrFunc = func(err error)

// Client an OCF local client.
type Client struct {
	tlsConfig              *TLSConfig
	errFunc                ErrFunc
	dialFunc               DialFunc
	discoveryConfiguration DiscoveryConfiguration
}

func checkTLSConfig(cfg *TLSConfig) *TLSConfig {
	if cfg == nil {
		cfg = new(TLSConfig)
	}
	if cfg.GetCertificate == nil {
		cfg.GetCertificate = func() (tls.Certificate, error) {
			return tls.Certificate{}, MakeUnimplemented(fmt.Errorf("not supported"))
		}
	}
	if cfg.GetCertificateAuthorities == nil {
		cfg.GetCertificateAuthorities = func() ([]*x509.Certificate, error) {
			return nil, MakeUnimplemented(fmt.Errorf("not supported"))
		}
	}
	return cfg
}

type config struct {
	tlsConfig              *TLSConfig
	errFunc                ErrFunc
	dialFunc               DialFunc
	discoveryConfiguration DiscoveryConfiguration
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
	MulticastAddressUDP4 []string // default: "[224.0.1.187:5683] (local.DiscoveryAddressUDP4), empty: don't use ipv4 multicast"
	MulticastAddressUDP6 []string // default: "[ff02::158]:5683", "[ff03::158]:5683", "[ff05::158]:5683]"] (local.DiscoveryAddressUDP6), empty: don't use ipv6 multicast"
}

// WithDiscoveryConfiguration override default DiscoveryConfiguration
func WithDiscoveryConfiguration(d DiscoveryConfiguration) OptionFunc {
	return func(cfg config) config {
		cfg.discoveryConfiguration = d
		return cfg
	}
}

func WithErr(errFunc ErrFunc) OptionFunc {
	return func(cfg config) config {
		if errFunc != nil {
			cfg.errFunc = errFunc
		}
		return cfg
	}
}

type DialFunc func(ctx context.Context, addr net.Addr, tlsConfig *TLSConfig) (*coap.ClientCloseHandler, error)

func WithDial(dial DialFunc) OptionFunc {
	return func(cfg config) config {
		if dial != nil {
			cfg.dialFunc = dial
		}
		return cfg
	}
}

func (c *Client) getDeviceConfiguration() deviceConfiguration {
	return deviceConfiguration{
		errFunc:                c.errFunc,
		discoveryConfiguration: c.discoveryConfiguration,
		dialFunc:               c.dialFunc,
		tlsConfig:              c.tlsConfig,
	}
}

func NewClient(opts ...OptionFunc) *Client {
	cfg := config{
		errFunc: func(err error) {
			log.Debug(err)
		},
		dialFunc: func(ctx context.Context, addr net.Addr, tlsConfig *TLSConfig) (*coap.ClientCloseHandler, error) {
			return DefaultDialFunc(ctx, addr, tlsConfig)
		},
		discoveryConfiguration: DiscoveryConfiguration{
			MulticastHopLimit:    2,
			MulticastAddressUDP4: DiscoveryAddressUDP4,
			MulticastAddressUDP6: DiscoveryAddressUDP6,
		},
	}
	for _, o := range opts {
		cfg = o(cfg)
	}

	cfg.tlsConfig = checkTLSConfig(cfg.tlsConfig)
	return &Client{
		dialFunc:               cfg.dialFunc,
		errFunc:                cfg.errFunc,
		discoveryConfiguration: cfg.discoveryConfiguration,
		tlsConfig:              cfg.tlsConfig,
	}
}
