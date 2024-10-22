package generateCertificate

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"

	ocfSigner "github.com/plgd-dev/device/v2/pkg/security/signer"
)

var (
	ASN1KeyUsage         = asn1.ObjectIdentifier{2, 5, 29, 15}
	ASN1BasicConstraints = asn1.ObjectIdentifier{2, 5, 29, 19}
	ASN1ExtKeyUsage      = asn1.ObjectIdentifier{2, 5, 29, 37}
)

// GenerateCSR creates CSR according to configuration.
func GenerateCSR(cfg Configuration, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	subj := cfg.ToPkixName()

	ips, err := cfg.ToIPAddresses()
	if err != nil {
		return nil, err
	}

	signatureAlgorithm, err := cfg.ToSignatureAlgorithm()
	if err != nil {
		return nil, err
	}

	extraExtensions := make([]pkix.Extension, 0, 3)
	if !cfg.BasicConstraints.Ignore {
		bcVal, errM := asn1.Marshal(BasicConstraints{false})
		if errM != nil {
			return nil, errM
		}
		extraExtensions = append(extraExtensions, pkix.Extension{
			Id:       ASN1BasicConstraints,
			Value:    bcVal,
			Critical: false,
		})
	}

	keyUsages, err := cfg.AsnKeyUsages()
	if err != nil {
		return nil, err
	}
	if keyUsages.BitLength > 0 {
		val, errM := asn1.Marshal(keyUsages)
		if errM != nil {
			return nil, errM
		}
		extraExtensions = append(extraExtensions, pkix.Extension{
			Id:       ASN1KeyUsage,
			Value:    val,
			Critical: false,
		})
	}

	extensionKeyUsages, err := cfg.AsnExtensionKeyUsages()
	if err != nil {
		return nil, err
	}
	if len(extensionKeyUsages) > 0 {
		val, errM := asn1.Marshal(extensionKeyUsages)
		if errM != nil {
			return nil, errM
		}
		extraExtensions = append(extraExtensions, pkix.Extension{
			Id:       ASN1ExtKeyUsage,
			Value:    val,
			Critical: false,
		})
	}

	rawSubj := subj.ToRDNSequence()
	asn1Subj, _ := asn1.Marshal(rawSubj)
	template := x509.CertificateRequest{
		RawSubject:         asn1Subj,
		DNSNames:           cfg.SubjectAlternativeName.DNSNames,
		IPAddresses:        ips,
		SignatureAlgorithm: signatureAlgorithm,
	}
	if len(extraExtensions) > 0 {
		template.ExtraExtensions = extraExtensions
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}), nil
}

func GenerateCert(cfg Configuration, privateKey *ecdsa.PrivateKey, signerCA []*x509.Certificate, signerCAKey *ecdsa.PrivateKey) ([]byte, error) {
	csr, err := GenerateCSR(cfg, privateKey)
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
	s, err := ocfSigner.NewBasicCertificateSigner(signerCA, signerCAKey, notBefore, notAfter, crlDistributionPoints)
	if err != nil {
		return nil, err
	}
	return s.Sign(context.Background(), csr)
}
