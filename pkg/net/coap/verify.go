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

package coap

import (
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var ExtendedKeyUsage_IDENTITY_CERTIFICATE = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 44924, 1, 6}

func verifyOcfEKU(cert *x509.Certificate) error {
	hasOcfID := false
	for _, eku := range cert.UnknownExtKeyUsage {
		if eku.Equal(ExtendedKeyUsage_IDENTITY_CERTIFICATE) {
			hasOcfID = true
			break
		}
	}
	if !hasOcfID {
		return errors.New("certificate does not contain ExtKeyUsage with OCF ID(1.3.6.1.4.1.44924.1.6")
	}
	return nil
}

func verifyEKU(cert *x509.Certificate, requireClient, requireServer, requireOcfId bool) error {
	hasClient := false
	hasServer := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageClientAuth {
			hasClient = true
			continue
		}
		if eku == x509.ExtKeyUsageServerAuth {
			hasServer = true
			continue
		}
	}
	if requireClient && !hasClient {
		return errors.New("certificate does not contain ExtKeyUsageClientAuth")
	}
	if requireServer && !hasServer {
		return errors.New("certificate does not contain ExtKeyUsageServerAuth")
	}

	if !requireOcfId {
		return nil
	}
	return verifyOcfEKU(cert)
}

func getUUIDFromSubjectCommonName(cert *x509.Certificate) (uuid.UUID, error) {
	cn := strings.Split(cert.Subject.CommonName, ":")
	if len(cn) != 2 {
		return uuid.UUID{}, fmt.Errorf("invalid subject common name: %v", cert.Subject.CommonName)
	}
	if strings.ToLower(cn[0]) != "uuid" {
		return uuid.UUID{}, fmt.Errorf("invalid subject common name %v: 'uuid' - not found", cert.Subject.CommonName)
	}
	id, err := uuid.Parse(cn[1])
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid subject common name %v: %w", cert.Subject.CommonName, err)
	}
	return id, nil
}

func GetDeviceIDFromIdentityCertificate(cert *x509.Certificate) (string, error) {
	// verify EKU manually
	if err := verifyEKU(cert, true, false, true); err != nil {
		return "", err
	}
	deviceID, err := getUUIDFromSubjectCommonName(cert)
	if err != nil {
		return "", err
	}
	return deviceID.String(), nil
}

func VerifyIdentityCertificate(cert *x509.Certificate) error {
	if err := verifyEKU(cert, true, true, false); err != nil {
		return err
	}
	if _, err := GetDeviceIDFromIdentityCertificate(cert); err != nil {
		return err
	}
	return nil
}

func VerifyCloudCertificate(cert *x509.Certificate, cloudID uuid.UUID) error {
	if err := verifyEKU(cert, true, true, false); err != nil {
		return err
	}
	id, err := getUUIDFromSubjectCommonName(cert)
	if err != nil {
		return err
	}

	if id != cloudID {
		return fmt.Errorf("invalid cloud certificate: invalid cloudID(%v)", id.String())
	}
	return nil
}
