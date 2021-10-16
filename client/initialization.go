package client

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	kitSecurity "github.com/plgd-dev/kit/v2/security"
	"github.com/plgd-dev/kit/v2/security/generateCertificate"
)

func generateSDKCertificate(ctx context.Context, csr []byte, sign SignFunc, priv *ecdsa.PrivateKey) (tls.Certificate, []*x509.Certificate, error) {
	cert, err := sign(ctx, csr)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("cannot sign csr: %w", err)
	}
	derKey, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("cannot marhsal private key: %w", err)
	}
	key := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: derKey})

	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("cannot create tls certificate: %w", err)
	}

	certsFromChain, err := kitSecurity.ParseX509FromPEM(cert)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("cannot parse cert chain: %w", err)
	}

	return tlsCert, []*x509.Certificate{certsFromChain[len(certsFromChain)-1]}, nil
}

func GenerateSDKIdentityCertificate(ctx context.Context, sign SignFunc, sdkDeviceID string) (tls.Certificate, []*x509.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("cannot generate private key: %w", err)
	}
	csr, err := generateCertificate.GenerateIdentityCSR(generateCertificate.Configuration{}, sdkDeviceID, priv)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("cannot generate identity csr: %w", err)
	}
	return generateSDKCertificate(ctx, csr, sign, priv)
}

func GenerateSDKManufacturerCertificate(ctx context.Context, sign SignFunc, ID string) (tls.Certificate, []*x509.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("cannot generate private key: %w", err)
	}
	cfg := generateCertificate.Configuration{}
	cfg.Subject.CommonName = "Manufacturer certificate for" + ID
	cfg.ExtensionKeyUsages = []string{"client"}
	csr, err := generateCertificate.GenerateCSR(cfg, priv)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("cannot generate identity csr: %w", err)
	}
	return generateSDKCertificate(ctx, csr, sign, priv)
}

func (c *Client) Initialization(ctx context.Context) (err error) {
	return c.deviceOwner.Initialization(ctx)
}

// GetIdentityCertificate returns certificate for connection
func (c *Client) GetIdentityCertificate() (tls.Certificate, error) {
	return c.deviceOwner.GetIdentityCertificate()
}

func (c *Client) GetIdentityCACerts() ([]*x509.Certificate, error) {
	return c.deviceOwner.GetIdentityCACerts()
}
