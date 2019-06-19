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
	"github.com/go-ocf/sdk/local/resource"
)

var testCfg = ocf.Config{
	Protocol: "tcp",
	Resource: resource.Config{
		ResourceHrefExpiration: time.Hour,
		DiscoveryTimeout:       time.Second * 3,
		DiscoveryDelay:         100 * time.Millisecond,

		Errors: func(error) {},
	},
}

type Client struct {
	*ocf.Client
	otm *ocf.ManufacturerOTMClient

	DeviceID string
}

func NewTestSecureClient() (*Client, error) {
	cert, err := tls.X509KeyPair(CertPEMBlock, KeyPEMBlock)
	if err != nil {
		return nil, err
	}
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

	testOwnCfg := testCfg
	testOwnCfg.TLSConfig.GetCertificate = func() (tls.Certificate, error) {
		return cert, nil
	}
	testOwnCfg.TLSConfig.GetCertificateAuthorities = func() ([]*x509.Certificate, error) {
		return []*x509.Certificate{ca}, nil
	}

	signer := ocf.NewBasicCertificateSigner(ca, caKey, time.Hour*86400)
	otm := ocf.NewManufacturerOTMClient(cert, ca, signer, []*x509.Certificate{ca})
	if err != nil {
		return nil, err
	}

	c, err := ocf.NewClientFromConfig(testOwnCfg, nil)
	if err != nil {
		return nil, err
	}

	return &Client{Client: c, otm: otm}, nil
}

func (c *Client) SetUpTestDevice(t *testing.T) {
	id := testGetDeviceID(t, c.Client, true)

	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := c.OwnDevice(timeout, id, c.otm)
	require.NoError(t, err)
	c.DeviceID = id
}

func (c *Client) Close() {
	if c.DeviceID == "" {
		return
	}
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := c.DisownDevice(timeout, c.DeviceID)
	if err != nil {
		panic(err)
	}
}
