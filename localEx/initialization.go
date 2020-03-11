package localEx

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/go-ocf/kit/security/generateCertificate"
	ocf "github.com/go-ocf/sdk/local"
)

func GenerateSDKIdentityCertificate(ctx context.Context, signer ocf.CertificateSigner, sdkDeviceID string) (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("cannot generate private key: %w", err)
	}
	csr, err := generateCertificate.GenerateIdentityCSR(generateCertificate.Configuration{}, sdkDeviceID, priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("cannot generate identity csr: %w", err)
	}
	cert, err := signer.Sign(ctx, csr)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("cannot sign csr: %w", err)
	}
	derKey, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("cannot marhsal private key: %w", err)
	}
	key := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: derKey})

	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("cannot create tls certificate: %w", err)
	}
	return tlsCert, nil
}

func (c *Client) Initialization(ctx context.Context) (err error) {
	return c.deviceOwner.Initialization(ctx)
}

// GetIdentityCertificate returns certificate for connection
func (c *Client) GetIdentityCertificate() (tls.Certificate, error) {
	return c.deviceOwner.GetIdentityCertificate()
}

// GetAccessTokenURL returns access token url.
func (c *Client) GetAccessTokenURL(ctx context.Context) (string, error) {
	return c.deviceOwner.GetAccessTokenURL(ctx)
}

// GetOnboardAuthorizationCodeURL returns access auth code url.
func (c *Client) GetOnboardAuthorizationCodeURL(ctx context.Context, deviceID string) (string, error) {
	return c.deviceOwner.GetOnboardAuthorizationCodeURL(ctx, deviceID)
}
