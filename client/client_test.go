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

package client_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/require"
)

const TestTimeout = time.Second * 8

type testSetupSecureClient struct {
	ca      []*x509.Certificate
	mfgCA   []*x509.Certificate
	mfgCert tls.Certificate
}

func (c *testSetupSecureClient) GetManufacturerCertificate() (tls.Certificate, error) {
	if c.mfgCert.PrivateKey == nil {
		return c.mfgCert, fmt.Errorf("private key not set")
	}
	return c.mfgCert, nil
}

func (c *testSetupSecureClient) GetManufacturerCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.mfgCA) == 0 {
		return nil, fmt.Errorf("certificate authority not set")
	}
	return c.mfgCA, nil
}

func (c *testSetupSecureClient) GetRootCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.ca) == 0 {
		return nil, fmt.Errorf("certificate authorities not set")
	}
	return c.ca, nil
}

func NewTestSecureClient() (*client.Client, error) {
	return newTestSecureClient(test.IdentityIntermediateCA, test.IdentityIntermediateCAKey)
}

func NewTestSecureClientWithGeneratedCertificate() (*client.Client, error) {
	var cfgCA generateCertificate.Configuration
	cfgCA.Subject.CommonName = "anotherClient"
	cfgCA.ValidFrom = "now"
	cfgCA.ValidFor = time.Hour

	priv, err := cfgCA.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("cannot generate private key: %w", err)
	}
	cert, err := generateCertificate.GenerateRootCA(cfgCA, priv)
	if err != nil {
		return nil, fmt.Errorf("cannot generate root ca: %w", err)
	}
	derKey, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("cannot marhsal private key: %w", err)
	}
	key := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: derKey})
	return newTestSecureClient(cert, key)
}

func newTestSecureClient(signerCert, signerKey []byte) (*client.Client, error) {
	cfg := client.Config{
		DeviceOwnershipSDK: &client.DeviceOwnershipSDKConfig{
			ID:               CertIdentity,
			Cert:             string(signerCert),
			CertKey:          string(signerKey),
			CreateSignerFunc: test.NewIdentityCertificateSigner,
		},
	}
	mfgTrustedCABlock, _ := pem.Decode(test.RootCACrt)
	if mfgTrustedCABlock == nil {
		return nil, fmt.Errorf("mfgTrustedCABlock is empty")
	}
	mfgCA, err := x509.ParseCertificates(mfgTrustedCABlock.Bytes)
	if err != nil {
		return nil, err
	}
	mfgCert, err := tls.X509KeyPair(test.MfgCert, test.MfgKey)
	if err != nil {
		return nil, fmt.Errorf("cannot X509KeyPair: %w", err)
	}

	client, err := client.NewClientFromConfig(&cfg, &testSetupSecureClient{
		mfgCA:   mfgCA,
		mfgCert: mfgCert,
	}, core.NewNilLogger(),
	)
	if err != nil {
		return nil, err
	}
	err = client.Initialization(context.Background())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func disown(t *testing.T, c *client.Client, deviceID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	err := c.DisownDevice(ctx, deviceID)
	require.NoError(t, err)
}

var CertIdentity = "00000000-0000-0000-0000-000000000001"
