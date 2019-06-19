package local

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"math/big"
	"time"

	kitNetCoap "github.com/go-ocf/kit/net/coap"
)

type CertificateSigner interface {
	//csr is encoded by DER
	Sign(ctx context.Context, csr []byte) ([]byte, error)
}

type BasicCertificateSigner struct {
	caCert   *x509.Certificate
	caKey    *ecdsa.PrivateKey
	validFor time.Duration
}

func NewBasicCertificateSigner(caCert *x509.Certificate, caKey *ecdsa.PrivateKey, validFor time.Duration) *BasicCertificateSigner {
	return &BasicCertificateSigner{caCert: caCert, caKey: caKey, validFor: validFor}
}

func (s *BasicCertificateSigner) Sign(ctx context.Context, csr []byte) (signedCsr []byte, err error) {
	certificateRequest, err := x509.ParseCertificateRequest(csr)
	if err != nil {
		return
	}

	err = certificateRequest.CheckSignature()
	if err != nil {
		return
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(s.validFor)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	template := x509.Certificate{
		SerialNumber:       serialNumber,
		NotBefore:          notBefore,
		NotAfter:           notAfter,
		Subject:            certificateRequest.Subject,
		PublicKeyAlgorithm: certificateRequest.PublicKeyAlgorithm,
		PublicKey:          certificateRequest.PublicKey,
		SignatureAlgorithm: certificateRequest.SignatureAlgorithm,
		DNSNames:           certificateRequest.DNSNames,
		IPAddresses:        certificateRequest.IPAddresses,
		Extensions:         certificateRequest.Extensions,
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		UnknownExtKeyUsage: []asn1.ObjectIdentifier{kitNetCoap.ExtendedKeyUsage_IDENTITY_CERTIFICATE},
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	signedCsr, err = x509.CreateCertificate(rand.Reader, &template, s.caCert, certificateRequest.PublicKey, s.caKey)
	return
}
