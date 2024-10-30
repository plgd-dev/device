package signer

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"math/big"
	"time"

	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
)

type BasicCertificateSigner struct {
	caCert                []*x509.Certificate
	caKey                 crypto.PrivateKey
	validNotBefore        time.Time
	validNotAfter         time.Time
	crlDistributionPoints []string
}

func NewBasicCertificateSigner(caCert []*x509.Certificate, caKey crypto.PrivateKey, validNotBefore, validNotAfter time.Time, crlDistributionPoints []string) (*BasicCertificateSigner, error) {
	if err := pkgX509.ValidateCRLDistributionPoints(crlDistributionPoints); err != nil {
		return nil, err
	}
	return &BasicCertificateSigner{
		caCert:                caCert,
		caKey:                 caKey,
		validNotBefore:        validNotBefore,
		validNotAfter:         validNotAfter,
		crlDistributionPoints: crlDistributionPoints,
	}, nil
}

func (s *BasicCertificateSigner) Sign(_ context.Context, csr []byte) ([]byte, error) {
	certificateRequest, err := pkgX509.ParseAndCheckCertificateRequest(csr)
	if err != nil {
		return nil, err
	}

	if len(s.caCert) == 0 {
		return nil, errors.New("cannot sign with empty signer CA certificates")
	}
	notBefore := s.validNotBefore
	notAfter := s.validNotAfter
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		Subject:               certificateRequest.Subject,
		PublicKeyAlgorithm:    certificateRequest.PublicKeyAlgorithm,
		PublicKey:             certificateRequest.PublicKey,
		SignatureAlgorithm:    s.caCert[0].SignatureAlgorithm,
		DNSNames:              certificateRequest.DNSNames,
		IPAddresses:           certificateRequest.IPAddresses,
		URIs:                  certificateRequest.URIs,
		EmailAddresses:        certificateRequest.EmailAddresses,
		ExtraExtensions:       certificateRequest.Extensions,
		CRLDistributionPoints: s.crlDistributionPoints,
	}
	signedCsr, err := x509.CreateCertificate(rand.Reader, &template, s.caCert[0], certificateRequest.PublicKey, s.caKey)
	if err != nil {
		return nil, err
	}
	return pkgX509.CreatePemChain(s.caCert, signedCsr)
}
