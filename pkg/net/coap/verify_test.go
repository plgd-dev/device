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

package coap_test

import (
	"crypto/ecdsa"
	"crypto/x509"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/pkg/security/generateCertificate"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/stretchr/testify/require"
)

func generateRootCA(t *testing.T, cfg generateCertificate.Configuration) ([]*x509.Certificate, *ecdsa.PrivateKey) {
	key, err := cfg.GenerateKey()
	require.NoError(t, err)
	pem, err := generateCertificate.GenerateRootCA(cfg, key)
	require.NoError(t, err)
	certs, err := pkgX509.ParsePemCertificates(pem)
	require.NoError(t, err)
	return certs, key
}

func TestVerifyIdentityCertificate(t *testing.T) {
	cfg := generateCertificate.Configuration{
		ValidFor: time.Minute,
	}
	rootCA, rootCAKey := generateRootCA(t, cfg)

	key, err := cfg.GenerateKey()
	require.NoError(t, err)
	id := uuid.New().String()
	certPem, err := generateCertificate.GenerateIdentityCert(cfg, id, key, rootCA, rootCAKey)
	require.NoError(t, err)
	cert, err := pkgX509.ParsePemCertificates(certPem)
	require.NoError(t, err)

	err = coap.VerifyIdentityCertificate(cert[0])
	require.NoError(t, err)
}

func TestVerifyCloudCertificate(t *testing.T) {
	cfg := generateCertificate.Configuration{
		ValidFor: time.Minute,
	}
	rootCA, rootCAKey := generateRootCA(t, cfg)

	key, err := cfg.GenerateKey()
	require.NoError(t, err)
	cloudID := uuid.New()
	certPem, err := generateCertificate.GenerateIdentityCert(cfg, cloudID.String(), key, rootCA, rootCAKey)
	require.NoError(t, err)
	cert, err := pkgX509.ParsePemCertificates(certPem)
	require.NoError(t, err)

	err = coap.VerifyCloudCertificate(cert[0], cloudID)
	require.NoError(t, err)
}
