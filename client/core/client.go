// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package core

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"

	"github.com/pion/dtls/v2"
	"github.com/pion/logging"
	pkgError "github.com/plgd-dev/device/v2/pkg/error"
	"github.com/plgd-dev/device/v2/pkg/log"
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

type Config struct {
	TLSConfig *TLSConfig
	Logger    Logger
	DialDTLS  DialDTLS
	DialTLS   DialTLS
	DialTCP   DialTCP
	DialUDP   DialUDP
}

type OptionFunc func(Config) Config

func WithTLS(tlsConfig *TLSConfig) OptionFunc {
	return func(cfg Config) Config {
		if tlsConfig != nil {
			cfg.TLSConfig = tlsConfig
		}
		return cfg
	}
}

// DiscoveryConfiguration setup discovery configuration
type DiscoveryConfiguration struct {
	MulticastHopLimit    int      // default: {224.0.1.187:5683, ff02::158]:5683} = 1, {[ff03::158]:5683, [ff05::158]:5683} = 255, https://openconnectivity.org/specs/OCF_Core_Specification.pdf 12.2.9
	MulticastAddressUDP4 []string // default: "[224.0.1.187:5683] (client.DiscoveryAddressUDP4), empty: don't use ipv4 multicast"
	MulticastAddressUDP6 []string // default: "[ff02::158]:5683", "[ff03::158]:5683", "[ff05::158]:5683]"] (client.DiscoveryAddressUDP6), empty: don't use ipv6 multicast"
	MulticastOptions     []coapNet.MulticastOption
}

func WithLogger(logger Logger) OptionFunc {
	return func(cfg Config) Config {
		if logger != nil {
			cfg.Logger = logger
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
	return func(cfg Config) Config {
		if dial != nil {
			cfg.DialDTLS = dial
		}
		return cfg
	}
}

func WithDialTLS(dial DialTLS) OptionFunc {
	return func(cfg Config) Config {
		if dial != nil {
			cfg.DialTLS = dial
		}
		return cfg
	}
}

func WithDialTCP(dial DialTCP) OptionFunc {
	return func(cfg Config) Config {
		if dial != nil {
			cfg.DialTCP = dial
		}
		return cfg
	}
}

func WithDialUDP(dial DialUDP) OptionFunc {
	return func(cfg Config) Config {
		if dial != nil {
			cfg.DialUDP = dial
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
		MulticastHopLimit:    0, // will be set to 1 or 255 based on address
		MulticastAddressUDP4: DiscoveryAddressUDP4,
		MulticastAddressUDP6: DiscoveryAddressUDP6,
		MulticastOptions: []coapNet.MulticastOption{coapNet.WithMulticastInterfaceError(func(iface *net.Interface, err error) {
			// ignore error
		})},
	}
}

func NewClient(opts ...OptionFunc) *Client {
	cfg := Config{
		Logger:   logging.NewDefaultLoggerFactory().NewLogger("client"),
		DialDTLS: coap.DialUDPSecure,
		DialTLS:  coap.DialTCPSecure,
		DialTCP:  coap.DialTCP,
		DialUDP:  coap.DialUDP,
	}
	for _, o := range opts {
		cfg = o(cfg)
	}
	if cfg.Logger == nil {
		cfg.Logger = log.NewNilLogger()
	}

	cfg.TLSConfig = checkTLSConfig(cfg.TLSConfig)
	return &Client{
		dialDTLS:  cfg.DialDTLS,
		dialTLS:   cfg.DialTLS,
		dialTCP:   cfg.DialTCP,
		dialUDP:   cfg.DialUDP,
		logger:    cfg.Logger,
		tlsConfig: cfg.TLSConfig,
	}
}
