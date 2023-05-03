package test

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/kit/v2/security"
)

func NewTestSigner() (core.CertificateSigner, error) {
	identityIntermediateCA, err := security.ParseX509FromPEM(IdentityIntermediateCA)
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

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 86400)
	return NewIdentityCertificateSigner(identityIntermediateCA, identityIntermediateCAKey, notBefore, notAfter), nil
}
