/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package x509_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

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

func TestParsePemCertificates(t *testing.T) {
	generateRootCA(t, generateCertificate.Configuration{})
}

func TestParsePemCertificatesChain(t *testing.T) {
	cfg := generateCertificate.Configuration{}
	ca, caKey := generateRootCA(t, generateCertificate.Configuration{})

	key, err := cfg.GenerateKey()
	require.NoError(t, err)
	got, err := generateCertificate.GenerateIntermediateCA(cfg, key, ca, caKey)
	require.NoError(t, err)
	_, err = pkgX509.ParsePemCertificates(got)
	require.NoError(t, err)
}

func encodeKeyToPem(t *testing.T, key *ecdsa.PrivateKey) []byte {
	b, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	block := &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	pem := pem.EncodeToMemory(block)
	return pem
}

func TestParsePemCertificates_Fail(t *testing.T) {
	// invalid input
	_, err := pkgX509.ParsePemCertificates(nil)
	require.Error(t, err)

	// pem encoded key instead of certificate
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	pem := encodeKeyToPem(t, key)
	_, err = pkgX509.ParsePemCertificates(pem)
	require.Error(t, err)
}

func TestReadPemCertificates(t *testing.T) {
	cfg := generateCertificate.Configuration{}
	key, err := cfg.GenerateKey()
	require.NoError(t, err)
	pem, err := generateCertificate.GenerateRootCA(cfg, key)
	require.NoError(t, err)

	testFilePath := "./test.pem"
	err = os.WriteFile(testFilePath, pem, 0o600)
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(testFilePath)
	}()

	_, err = pkgX509.ReadPemCertificates(testFilePath)
	require.NoError(t, err)
}

func TestReadPemCertificates_Fail(t *testing.T) {
	_, err := pkgX509.ReadPemCertificates("")
	require.Error(t, err)
}

func TestParsePemEcdsaPrivateKey(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	keyPem := encodeKeyToPem(t, key)
	_, err = pkgX509.ParsePemEcdsaPrivateKey(keyPem)
	require.NoError(t, err)

	pkcsDer, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	block := &pem.Block{Type: "EC PRIVATE KEY", Bytes: pkcsDer}
	pkcsPem := pem.EncodeToMemory(block)
	_, err = pkgX509.ParsePemEcdsaPrivateKey(pkcsPem)
	require.NoError(t, err)
}

func TestParsePemEcdsaPrivateKey_Fail(t *testing.T) {
	_, err := pkgX509.ParsePemEcdsaPrivateKey(nil)
	require.Error(t, err)

	cfg := generateCertificate.Configuration{}
	ecKey, err := cfg.GenerateKey()
	require.NoError(t, err)
	ecPem, err := generateCertificate.GenerateRootCA(cfg, ecKey)
	require.NoError(t, err)
	_, err = pkgX509.ParsePemEcdsaPrivateKey(ecPem)
	require.Error(t, err)

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pkcsDer, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	require.NoError(t, err)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: pkcsDer}
	pkcsPem := pem.EncodeToMemory(block)
	_, err = pkgX509.ParsePemEcdsaPrivateKey(pkcsPem)
	require.Error(t, err)
}

func TestReadPemEcdsaPrivateKey(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	keyPem := encodeKeyToPem(t, key)

	testFilePath := "./test.pem"
	err = os.WriteFile(testFilePath, keyPem, 0o600)
	require.NoError(t, err)
	defer func() {
		_ = os.Remove(testFilePath)
	}()

	_, err = pkgX509.ReadPemEcdsaPrivateKey(testFilePath)
	require.NoError(t, err)
}

func TestReadPemEcdsaPrivateKey_Fail(t *testing.T) {
	_, err := pkgX509.ReadPemEcdsaPrivateKey("")
	require.Error(t, err)
}

func TestParseCertificates_Fail(t *testing.T) {
	_, err := pkgX509.ParseCertificates(&tls.Certificate{})
	require.Error(t, err)
}

func generateValidCSR(t *testing.T) []byte {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "failed to generate private key")

	template := &x509.CertificateRequest{
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, template, priv)
	require.NoError(t, err, "failed to create certificate request")

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
}

func generateCSRWithInvalidSignature(t *testing.T) []byte {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "failed to generate private key")

	template := &x509.CertificateRequest{
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, template, priv)
	require.NoError(t, err, "failed to create certificate request")

	// Tamper with the CSR bytes to invalidate the signature
	csrBytes[len(csrBytes)-1] ^= 0xFF

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
}

func TestParseAndCheckCertificateRequest(t *testing.T) {
	tests := []struct {
		name    string
		csr     []byte
		wantErr bool
	}{
		{
			name: "Valid CSR",
			csr:  generateValidCSR(t),
		},
		{
			name:    "Invalid PEM",
			csr:     []byte("invalid-pem-data"),
			wantErr: true,
		},
		{
			name:    "Invalid CSR",
			csr:     pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: []byte("invalid-csr-data")}),
			wantErr: true,
		},
		{
			name:    "Invalid Signature",
			csr:     generateCSRWithInvalidSignature(t),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pkgX509.ParseAndCheckCertificateRequest(tt.csr)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
