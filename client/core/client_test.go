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

package core_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/client/core/otm"
	justworks "github.com/plgd-dev/device/v2/client/core/otm/just-works"
	"github.com/plgd-dev/device/v2/client/core/otm/manufacturer"
	pkgError "github.com/plgd-dev/device/v2/pkg/error"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/test"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/go-coap/v3/udp"
	"github.com/stretchr/testify/require"
)

type Client struct {
	*core.Client
	mfgOtm       *manufacturer.Client
	justWorksOtm *justworks.Client

	DeviceID string
	*core.Device
	DeviceLinks schema.ResourceLinks
}

func NewTestSecureClient() (*Client, error) {
	identityCert := test.GenerateIdentityCert(CertIdentity)
	return NewTestSecureClientWithCert(identityCert, false, false)
}

func NewTestSecureClientWithTLS(disableDTLS, disableTCPTLS bool) (*Client, error) {
	identityCert := test.GenerateIdentityCert(CertIdentity)
	return NewTestSecureClientWithCert(identityCert, disableDTLS, disableTCPTLS)
}

func NewTestSecureClientWithCert(cert tls.Certificate, disableDTLS, disableTCPTLS bool) (*Client, error) {
	mfgCert, err := tls.X509KeyPair(test.MfgCert, test.MfgKey)
	if err != nil {
		return nil, err
	}

	mfgCa, err := pkgX509.ParsePemCertificates(test.RootCACrt)
	if err != nil {
		return nil, err
	}

	identityIntermediateCA, err := pkgX509.ParsePemCertificates(test.IdentityIntermediateCA)
	if err != nil {
		return nil, err
	}

	var manOpts []manufacturer.OptionFunc
	if disableDTLS {
		manOpts = append(manOpts, manufacturer.WithDialDTLS(func(context.Context, string, *dtls.Config, ...udp.Option) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}
	if disableTCPTLS {
		manOpts = append(manOpts, manufacturer.WithDialTLS(func(context.Context, string, *tls.Config, ...tcp.Option) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}

	mfgOtm := manufacturer.NewClient(mfgCert, mfgCa, manOpts...)
	justWorksOtm := justworks.NewClient()

	var opts []core.OptionFunc
	if disableDTLS {
		opts = append(opts, core.WithDialDTLS(func(context.Context, string, *dtls.Config, ...udp.Option) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}
	if disableTCPTLS {
		opts = append(opts, core.WithDialTLS(func(context.Context, string, *tls.Config, ...tcp.Option) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}
	opts = append(opts, core.WithTLS(&core.TLSConfig{
		GetCertificate: func() (tls.Certificate, error) {
			return cert, nil
		},
		GetCertificateAuthorities: func() ([]*x509.Certificate, error) {
			return identityIntermediateCA, nil
		},
	}),
	)

	c := core.NewClient(opts...)

	return &Client{Client: c, mfgOtm: mfgOtm, justWorksOtm: justWorksOtm}, nil
}

func (c *Client) SetUpTestDevice(t *testing.T) {
	signer, err := test.NewTestSigner()
	require.NoError(t, err)

	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)

	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	device, err := c.GetDeviceByMulticast(timeout, secureDeviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	eps := device.GetEndpoints()
	links, err := device.GetResourceLinks(timeout, eps)
	require.NoError(t, err)
	err = device.Own(timeout, links, []otm.Client{c.mfgOtm}, core.WithSetupCertificates(signer.Sign))
	require.NoError(t, err)
	links, err = device.GetResourceLinks(timeout, eps)
	require.NoError(t, err)
	c.Device = device
	c.DeviceID = secureDeviceID
	c.DeviceLinks = links
}

func (c *Client) Close() error {
	if c.DeviceID == "" {
		return nil
	}
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := c.Disown(timeout, c.DeviceLinks)
	if err != nil {
		return err
	}
	return c.Device.Close(timeout)
}

var CertIdentity = "00000000-0000-0000-0000-000000000001"
