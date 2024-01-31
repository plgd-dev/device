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

package x509

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// ParsePemCertificates parses x509 certificates from PEM format
func ParsePemCertificates(pemBlock []byte) ([]*x509.Certificate, error) {
	data := pemBlock
	var cas []*x509.Certificate
	for {
		certDERBlock, tmp := pem.Decode(data)
		if certDERBlock == nil {
			return nil, fmt.Errorf("cannot decode pem block")
		}
		certs, err := x509.ParseCertificates(certDERBlock.Bytes)
		if err != nil {
			return nil, err
		}
		cas = append(cas, certs...)
		if len(tmp) == 0 {
			break
		}
		data = tmp
	}
	return cas, nil
}

// ReadPemCertificates reads certificates from file in PEM format
func ReadPemCertificates(path string) ([]*x509.Certificate, error) {
	certPEMBlock, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	return ParsePemCertificates(certPEMBlock)
}

// ParsePemEcdsaPrivateKey parses private key from PEM format
func ParsePemEcdsaPrivateKey(pemBlock []byte) (*ecdsa.PrivateKey, error) {
	derBlock, _ := pem.Decode(pemBlock)
	if derBlock == nil {
		return nil, fmt.Errorf("cannot decode pem block")
	}

	if key, err := x509.ParsePKCS8PrivateKey(derBlock.Bytes); err == nil {
		switch key := key.(type) {
		case *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, fmt.Errorf("found unknown private key type in PKCS#8 wrapping")
		}
	}

	if key, err := x509.ParseECPrivateKey(derBlock.Bytes); err == nil {
		return key, nil
	}

	return nil, fmt.Errorf("failed to parse private key")
}

// ReadPemEcdsaPrivateKey loads private key from file in PEM format
func ReadPemEcdsaPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	certPEMBlock, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	return ParsePemEcdsaPrivateKey(certPEMBlock)
}

// ParseCertificates parses the CA chain certificates from the DER data.
func ParseCertificates(cert *tls.Certificate) ([]*x509.Certificate, error) {
	caChain := make([]*x509.Certificate, 0, 4)
	for _, derBytes := range cert.Certificate {
		ca, err := x509.ParseCertificates(derBytes)
		if err != nil {
			return nil, err
		}
		caChain = append(caChain, ca...)
	}
	if len(caChain) == 0 {
		return nil, fmt.Errorf("no certificates")
	}
	return caChain, nil
}
