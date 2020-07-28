package core

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/go-ocf/kit/net/coap"

	"github.com/go-ocf/kit/log"
)

// ErrFunc to log errors in goroutines
type ErrFunc = func(err error)

// Client an OCF local client.
type Client struct {
	tlsConfig              *TLSConfig
	errFunc                ErrFunc
	dialOptions            []coap.DialOptionFunc
	discoveryConfiguration DiscoveryConfiguration
	disableDTLS            bool
	disableTCPTLS          bool
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

type config struct {
	tlsConfig              *TLSConfig
	errFunc                ErrFunc
	dialOptions            []coap.DialOptionFunc
	discoveryConfiguration DiscoveryConfiguration
	disableTCPTLS          bool
	disableDTLS            bool
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

func WithoutTCPTLS() OptionFunc {
	return func(cfg config) config {
		cfg.disableTCPTLS = true
		return cfg
	}
}

func WithoutDTLS() OptionFunc {
	return func(cfg config) config {
		cfg.disableDTLS = true
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

func WithDialOptions(opts ...coap.DialOptionFunc) OptionFunc {
	return func(cfg config) config {
		if len(opts) > 0 {
			cfg.dialOptions = opts
		}
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

func (c *Client) getDeviceConfiguration() deviceConfiguration {
	return deviceConfiguration{
		tlsConfig:              c.tlsConfig,
		errFunc:                c.errFunc,
		dialOptions:            c.dialOptions,
		discoveryConfiguration: c.discoveryConfiguration,
		disableDTLS:            c.disableDTLS,
		disableTCPTLS:          c.disableTCPTLS,
	}
}

func NewClient(opts ...OptionFunc) *Client {
	cfg := config{
		errFunc: func(err error) {
			log.Debug(err)
		},
		dialOptions: []coap.DialOptionFunc{
			coap.WithDialDisablePeerTCPSignalMessageCSMs(),
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
	cfg.dialOptions = append(cfg.dialOptions, coap.WithErrors(cfg.errFunc))

	cfg.tlsConfig = checkTLSConfig(cfg.tlsConfig)
	return &Client{
		tlsConfig:              cfg.tlsConfig,
		errFunc:                cfg.errFunc,
		dialOptions:            cfg.dialOptions,
		discoveryConfiguration: cfg.discoveryConfiguration,
		disableDTLS:            cfg.disableDTLS,
		disableTCPTLS:          cfg.disableTCPTLS,
	}
}
