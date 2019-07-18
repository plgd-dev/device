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

	ocf "github.com/go-ocf/sdk/local"
)

type Client struct {
	*ocf.Client
	otm *ocf.ManufacturerOTMClient

	DeviceID string
	*ocf.Device
}

func NewTestSecureClient() (*Client, error) {
	cert, err := tls.X509KeyPair(CertPEMBlock, KeyPEMBlock)
	if err != nil {
		return nil, err
	}
	return NewTestSecureClientWithCert(cert)
}

func NewTestSecureClientWithCert(cert tls.Certificate) (*Client, error) {
	derBlock, _ := pem.Decode(CARootPemBlock)
	if derBlock == nil {
		return nil, fmt.Errorf("invalid CARootPemBlock")
	}
	ca, err := x509.ParseCertificate(derBlock.Bytes)
	if err != nil {
		return nil, err
	}
	derBlockKey, _ := pem.Decode(CARootKeyPemBlock)
	if derBlockKey == nil {
		return nil, fmt.Errorf("invalid CARootKeyPemBlock")
	}
	caKey, err := x509.ParseECPrivateKey(derBlockKey.Bytes)
	if err != nil {
		return nil, err
	}

	signer := ocf.NewBasicCertificateSigner(ca, caKey, time.Hour*86400)
	otm := ocf.NewManufacturerOTMClient(cert, ca, signer, []*x509.Certificate{ca})
	if err != nil {
		return nil, err
	}

	c := ocf.NewClient(ocf.WithTLS(&ocf.TLSConfig{
		GetCertificate: func() (tls.Certificate, error) {
			return cert, nil
		},
		GetCertificateAuthorities: func() ([]*x509.Certificate, error) {
			return []*x509.Certificate{ca}, nil
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
