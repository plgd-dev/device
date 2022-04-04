package client_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/test"
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
	cfg := client.Config{
		DeviceOwnershipSDK: &client.DeviceOwnershipSDKConfig{
			ID:               CertIdentity,
			Cert:             string(test.IdentityIntermediateCA),
			CertKey:          string(test.IdentityIntermediateCAKey),
			CreateSignerFunc: test.NewIdentityCertificateSigner,
		},
	}

	client, err := client.NewClientFromConfig(&cfg, &testSetupSecureClient{
		mfgCA:   mfgCA,
		mfgCert: mfgCert,
	}, func(err error) { fmt.Print(err) },
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

var (
	CertIdentity = "00000000-0000-0000-0000-000000000001"
)
