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

package manufacturer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/pion/dtls/v2"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/go-coap/v3/udp"
	kitNet "github.com/plgd-dev/kit/v2/net"
)

type (
	DialDTLS = func(ctx context.Context, addr string, dtlsCfg *dtls.Config, opts ...udp.Option) (*coap.ClientCloseHandler, error)
	DialTLS  = func(ctx context.Context, addr string, tlsCfg *tls.Config, opts ...tcp.Option) (*coap.ClientCloseHandler, error)
)

type Client struct {
	manufacturerCertificate tls.Certificate
	manufacturerCA          []*x509.Certificate
	dialDTLS                DialDTLS
	dialTLS                 DialTLS
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
	opts ...OptionFunc,
) *Client {
	c := Client{
		manufacturerCertificate: manufacturerCertificate,
		manufacturerCA:          manufacturerCA,
		dialDTLS:                coap.DialUDPSecure,
		dialTLS:                 coap.DialTCPSecure,
	}
	for _, o := range opts {
		c = o(c)
	}
	return &c
}

func (*Client) Type() doxm.OwnerTransferMethod {
	return doxm.ManufacturerCertificate
}

func (c *Client) Dial(ctx context.Context, addr kitNet.Addr) (*coap.ClientCloseHandler, error) {
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
			VerifyPeerCertificate: coap.NewVerifyPeerCertificate(rootCAs, func(*x509.Certificate) error { return nil }),
		}
		return c.dialDTLS(ctx, addr.String(), &tlsConfig)
	case schema.TCPSecureScheme:
		rootCAs := x509.NewCertPool()
		for _, ca := range c.manufacturerCA {
			rootCAs.AddCert(ca)
		}
		tlsConfig := tls.Config{
			InsecureSkipVerify:    true, //nolint:gosec
			Certificates:          []tls.Certificate{c.manufacturerCertificate},
			VerifyPeerCertificate: coap.NewVerifyPeerCertificate(rootCAs, func(*x509.Certificate) error { return nil }),
		}
		return c.dialTLS(ctx, addr.String(), &tlsConfig)
	}
	return nil, fmt.Errorf("cannot dial to url %v: scheme %v not supported", addr.URL(), addr.GetScheme())
}
