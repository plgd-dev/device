// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package signer

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/plgd-dev/device/v2/pkg/net/coap"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
)

type OCFIdentityCertificate struct {
	caCert                []*x509.Certificate
	caKey                 crypto.PrivateKey
	validNotBefore        time.Time
	validNotAfter         time.Time
	crlDistributionPoints []string
}

func NewOCFIdentityCertificate(caCert []*x509.Certificate, caKey crypto.PrivateKey, validNotBefore, validNotAfter time.Time, crlDistributionPoints []string) (*OCFIdentityCertificate, error) {
	if err := pkgX509.ValidateCRLDistributionPoints(crlDistributionPoints); err != nil {
		return nil, err
	}
	return &OCFIdentityCertificate{
		caCert:                caCert,
		caKey:                 caKey,
		validNotBefore:        validNotBefore,
		validNotAfter:         validNotAfter,
		crlDistributionPoints: crlDistributionPoints,
	}, nil
}

func (s *OCFIdentityCertificate) Sign(_ context.Context, csr []byte) ([]byte, error) {
	now := time.Now()
	notBefore := s.validNotBefore
	notAfter := s.validNotAfter
	for _, c := range s.caCert {
		if notBefore.Before(c.NotBefore) {
			notBefore = c.NotBefore
		}
		if notAfter.After(c.NotAfter) {
			notAfter = c.NotAfter
		}
	}
	if notBefore.After(notAfter) {
		return nil, fmt.Errorf("invalid time range: not before %v limit is greater than not after limit %v", notBefore.Format(time.RFC3339), notAfter.Format(time.RFC3339))
	}
	if now.Before(notBefore) {
		return nil, fmt.Errorf("not valid yet: current time %v is out of time range: %v <-> %v", now, notBefore.Format(time.RFC3339), notAfter.Format(time.RFC3339))
	}
	if now.After(notAfter) {
		return nil, fmt.Errorf("expired: current time %v is out of time range: %v <-> %v", now, notBefore.Format(time.RFC3339), notAfter.Format(time.RFC3339))
	}

	certificateRequest, err := pkgX509.ParseAndCheckCertificateRequest(csr)
	if err != nil {
		return nil, err
	}

	if len(s.caCert) == 0 {
		return nil, errors.New("cannot sign with empty signer CA certificates")
	}
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
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		UnknownExtKeyUsage:    []asn1.ObjectIdentifier{coap.ExtendedKeyUsage_IDENTITY_CERTIFICATE},
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		CRLDistributionPoints: s.crlDistributionPoints,
	}
	signedCsr, err := x509.CreateCertificate(rand.Reader, &template, s.caCert[0], certificateRequest.PublicKey, s.caKey)
	if err != nil {
		return nil, err
	}
	return pkgX509.CreatePemChain(s.caCert, signedCsr)
}
