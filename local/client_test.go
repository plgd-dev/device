package local_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	ocfSigner "github.com/go-ocf/kit/security/signer"
	ocf "github.com/go-ocf/sdk/local"
)

type Client struct {
	*ocf.Client
	otm *ocf.ManufacturerOTMClient

	DeviceID string
	*ocf.Device
}

func NewTestSecureClient() (*Client, error) {
	identityCert, err := tls.X509KeyPair(IdentityCert, IdentityKey)
	if err != nil {
		return nil, err
	}
	return NewTestSecureClientWithCert(identityCert)
}

func NewTestSecureClientWithCert(cert tls.Certificate) (*Client, error) {
	mfgCert, err := tls.X509KeyPair(MfgCert, MfgKey)
	if err != nil {
		return nil, err
	}
	mfgTrustedCABlock, _ := pem.Decode(MfgTrustedCA)
	if mfgTrustedCABlock == nil {
		return nil, fmt.Errorf("mfgTrustedCABlock is empty")
	}
	mfgCa, err := x509.ParseCertificate(mfgTrustedCABlock.Bytes)
	if err != nil {
		return nil, err
	}

	identityIntermediateCABlock, _ := pem.Decode(MfgTrustedCA)
	if identityIntermediateCABlock == nil {
		return nil, fmt.Errorf("identityIntermediateCABlock is empty")
	}
	identityIntermediateCA, err := x509.ParseCertificates(identityIntermediateCABlock.Bytes)
	if err != nil {
		return nil, err
	}
	identityIntermediateCAKeyBlock, _ := pem.Decode(IdentityIntermediateCAKey)
	if identityIntermediateCAKeyBlock == nil {
		return nil, fmt.Errorf("identityIntermediateCAKeyBlock is empty")
	}
	identityIntermediateCAKey, err := x509.ParseECPrivateKey(identityIntermediateCAKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	identityTrustedCABlock, _ := pem.Decode(IdentityTrustedCA)
	if identityTrustedCABlock == nil {
		return nil, fmt.Errorf("identityTrustedCABlock is empty")
	}
	identityTrustedCA, err := x509.ParseCertificates(identityTrustedCABlock.Bytes)
	if err != nil {
		return nil, err
	}

	signer := ocfSigner.NewIdentityCertificateSigner(identityIntermediateCA, identityIntermediateCAKey, time.Hour*86400)

	otm := ocf.NewManufacturerOTMClient(mfgCert, mfgCa, signer, identityTrustedCA)
	if err != nil {
		return nil, err
	}

	c := ocf.NewClient(ocf.WithTLS(&ocf.TLSConfig{
		GetCertificate: func() (tls.Certificate, error) {
			return cert, nil
		},
		GetCertificateAuthorities: func() ([]*x509.Certificate, error) {
			cas := identityTrustedCA
			cas = append(cas, mfgCa)
			return cas, nil
		},
	}))

	return &Client{Client: c, otm: otm}, nil
}

func (c *Client) SetUpTestDevice(t *testing.T) {
	id := testGetDeviceID(t, c.Client, true)

	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	device, _, err := c.GetDevice(timeout, id)
	require.NoError(t, err)
	err = device.Own(timeout, c.otm)
	require.NoError(t, err)
	c.Device = device
	c.DeviceID = id
}

func (c *Client) Close() {
	if c.DeviceID == "" {
		return
	}
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := c.Disown(timeout)
	if err != nil {
		panic(err)
	}
	c.Device.Close(timeout)
}
