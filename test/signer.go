package test

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
)

func NewTestSigner() (core.CertificateSigner, error) {
	identityIntermediateCA, err := pkgX509.ParsePemCertificates(IdentityIntermediateCA)
	if err != nil {
		return nil, err
	}
	identityIntermediateCAKeyBlock, _ := pem.Decode(IdentityIntermediateCAKey)
	if identityIntermediateCAKeyBlock == nil {
		return nil, errors.New("identityIntermediateCAKeyBlock is empty")
	}
	identityIntermediateCAKey, err := x509.ParseECPrivateKey(identityIntermediateCAKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 86400)
	return NewIdentityCertificateSigner(identityIntermediateCA, identityIntermediateCAKey, notBefore, notAfter, nil)
}
