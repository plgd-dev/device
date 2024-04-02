/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/pkg/log"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/test"
)

var CertIdentity = "00000000-0000-0000-0000-000000000001"

type testSetupSecureClient struct {
	ca      []*x509.Certificate
	mfgCA   []*x509.Certificate
	mfgCert tls.Certificate
}

func (c *testSetupSecureClient) GetManufacturerCertificate() (tls.Certificate, error) {
	if c.mfgCert.PrivateKey == nil {
		return c.mfgCert, errors.New("private key not set")
	}
	return c.mfgCert, nil
}

func (c *testSetupSecureClient) GetManufacturerCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.mfgCA) == 0 {
		return nil, errors.New("certificate authority not set")
	}
	return c.mfgCA, nil
}

func (c *testSetupSecureClient) GetRootCertificateAuthorities() ([]*x509.Certificate, error) {
	if len(c.ca) == 0 {
		return nil, errors.New("certificate authorities not set")
	}
	return c.ca, nil
}

func NewTestSetupSecureClient(ca, mfgCA []*x509.Certificate, mfgCert tls.Certificate) client.ApplicationCallback {
	return &testSetupSecureClient{
		ca:      ca,
		mfgCA:   mfgCA,
		mfgCert: mfgCert,
	}
}

func NewTestSecureClient() (*client.Client, error) {
	return newTestSecureClient(test.IdentityIntermediateCA, test.IdentityIntermediateCAKey, false)
}

func NewTestSecureClientWithBridgeSupport() (*client.Client, error) {
	return newTestSecureClient(test.IdentityIntermediateCA, test.IdentityIntermediateCAKey, true)
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
	return newTestSecureClient(cert, key, false)
}

func newTestSecureClient(signerCert, signerKey []byte, useDeviceIDInQuery bool) (*client.Client, error) {
	cfg := client.Config{
		DeviceOwnershipSDK: &client.DeviceOwnershipSDKConfig{
			ID:               CertIdentity,
			Cert:             string(signerCert),
			CertKey:          string(signerKey),
			CreateSignerFunc: test.NewIdentityCertificateSigner,
		},
		UseDeviceIDInQuery: useDeviceIDInQuery,
	}
	mfgTrustedCABlock, _ := pem.Decode(test.RootCACrt)
	if mfgTrustedCABlock == nil {
		return nil, errors.New("mfgTrustedCABlock is empty")
	}
	mfgCA, err := x509.ParseCertificates(mfgTrustedCABlock.Bytes)
	if err != nil {
		return nil, err
	}
	mfgCert, err := tls.X509KeyPair(test.MfgCert, test.MfgKey)
	if err != nil {
		return nil, fmt.Errorf("X509KeyPair failed: %w", err)
	}

	client, err := client.NewClientFromConfig(&cfg, &testSetupSecureClient{
		mfgCA:   mfgCA,
		mfgCert: mfgCert,
	}, log.NewStdLogger(log.LevelDebug))
	if err != nil {
		return nil, err
	}
	err = client.Initialization(context.Background())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func MakeMockDeviceResourcesObservationHandler() *MockDeviceResourcesObservationHandler {
	return &MockDeviceResourcesObservationHandler{
		res:   make(chan schema.ResourceLinks, 100),
		close: make(chan struct{}),
	}
}

type MockDeviceResourcesObservationHandler struct {
	res   chan schema.ResourceLinks
	close chan struct{}
}

func (h *MockDeviceResourcesObservationHandler) Handle(_ context.Context, body schema.ResourceLinks) {
	h.res <- body
}

func (h *MockDeviceResourcesObservationHandler) Error(err error) {
	fmt.Println(err)
}

func (h *MockDeviceResourcesObservationHandler) OnClose() {
	close(h.close)
}

func (h *MockDeviceResourcesObservationHandler) WaitForNotification(ctx context.Context) (schema.ResourceLinks, error) {
	select {
	case e := <-h.res:
		return e, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.close:
		return nil, errors.New("unexpected close")
	}
}

func (h *MockDeviceResourcesObservationHandler) WaitForClose(ctx context.Context) error {
	select {
	case e := <-h.res:
		return fmt.Errorf("unexpected notification %v", e)
	case <-ctx.Done():
		return ctx.Err()
	case <-h.close:
		return nil
	}
}

type MockResourceObservationHandler struct {
	Res   chan coap.DecodeFunc
	Close chan struct{}
}

func MakeMockResourceObservationHandler() *MockResourceObservationHandler {
	return &MockResourceObservationHandler{Res: make(chan coap.DecodeFunc, 10), Close: make(chan struct{})}
}

func (h *MockResourceObservationHandler) Handle(_ context.Context, body coap.DecodeFunc) {
	h.Res <- body
}

func (h *MockResourceObservationHandler) Error(err error) { fmt.Println(err) }

func (h *MockResourceObservationHandler) OnClose() { close(h.Close) }

func (h *MockResourceObservationHandler) WaitForNotification(ctx context.Context) (coap.DecodeFunc, error) {
	select {
	case e := <-h.Res:
		return e, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.Close:
		return nil, errors.New("unexpected close")
	}
}

func (h *MockResourceObservationHandler) WaitForClose(ctx context.Context) error {
	select {
	case e := <-h.Res:
		var d interface{}
		if err := e(d); err != nil {
			return fmt.Errorf("unexpected notification: cannot decode: %w", err)
		}
		return fmt.Errorf("unexpected notification %v", d)
	case <-ctx.Done():
		return ctx.Err()
	case <-h.Close:
		return nil
	}
}
