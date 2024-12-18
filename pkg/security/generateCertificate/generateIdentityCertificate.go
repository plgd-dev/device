package generateCertificate

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"

	ocfSigner "github.com/plgd-dev/device/v2/pkg/security/signer"
)

type BasicConstraints struct {
	CA bool
}

func NewIdentityCSRTemplate(deviceID string) (*x509.CertificateRequest, error) {
	subj := pkix.Name{
		CommonName: fmt.Sprintf("uuid:%v", deviceID),
	}

	val, err := asn1.Marshal([]asn1.ObjectIdentifier{{1, 3, 6, 1, 5, 5, 7, 3, 1}, {1, 3, 6, 1, 5, 5, 7, 3, 2}, {1, 3, 6, 1, 4, 1, 44924, 1, 6}})
	if err != nil {
		return nil, err
	}

	bcVal, err := asn1.Marshal(BasicConstraints{false})
	if err != nil {
		return nil, err
	}

	kuVal, err := asn1.Marshal(asn1.BitString{Bytes: []byte{1<<3 | 1<<7}, BitLength: 7}) // x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement
	if err != nil {
		return nil, err
	}

	template := x509.CertificateRequest{
		Subject: subj,
		ExtraExtensions: []pkix.Extension{
			{
				Id:       ASN1BasicConstraints,
				Value:    bcVal,
				Critical: false,
			},
			{
				Id:       ASN1KeyUsage,
				Value:    kuVal,
				Critical: false,
			},
			{
				Id:       ASN1ExtKeyUsage,
				Value:    val,
				Critical: false,
			},
		},
	}
	return &template, nil
}

// GenerateIdentityCSR creates identity CSR according to configuration.
func GenerateIdentityCSR(cfg Configuration, deviceID string, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	template, err := NewIdentityCSRTemplate(deviceID)
	if err != nil {
		return nil, err
	}
	signatureAlgorithm, err := cfg.ToSignatureAlgorithm()
	if err != nil {
		return nil, err
	}
	subj := cfg.ToPkixName()
	subj.CommonName = template.Subject.CommonName
	template.Subject = subj
	template.SignatureAlgorithm = signatureAlgorithm

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, privateKey)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}), nil
}

func GenerateIdentityCert(cfg Configuration, deviceID string, privateKey *ecdsa.PrivateKey, signerCA []*x509.Certificate, signerCAKey *ecdsa.PrivateKey) ([]byte, error) {
	csr, err := GenerateIdentityCSR(cfg, deviceID, privateKey)
	if err != nil {
		return nil, err
	}

	notBefore, err := cfg.ToValidFrom()
	if err != nil {
		return nil, err
	}
	notAfter := notBefore.Add(cfg.ValidFor)
	crlDistributionPoints, err := cfg.ToCRLDistributionPoints()
	if err != nil {
		return nil, err
	}
	s, err := ocfSigner.NewOCFIdentityCertificate(signerCA, signerCAKey, notBefore, notAfter, crlDistributionPoints)
	if err != nil {
		return nil, err
	}
	return s.Sign(context.Background(), csr)
}
