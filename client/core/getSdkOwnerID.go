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

package core

import (
	"crypto/x509"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
)

func getSdkError(err error) error {
	return fmt.Errorf("cannot get sdk id: %w", err)
}

func getSdkOwnerID(getCertificate GetCertificateFunc) (string, error) {
	if getCertificate == nil {
		return "", MakeUnimplemented(fmt.Errorf("getCertificate is not set"))
	}
	cert, err := getCertificate()
	if err != nil {
		return "", MakeInternal(getSdkError(err))
	}

	var errors *multierror.Error

	for _, c := range cert.Certificate {
		x509cert, err := x509.ParseCertificate(c)
		if err != nil {
			errors = multierror.Append(errors, err)
			continue
		}
		id, err := coap.GetDeviceIDFromIdentityCertificate(x509cert)
		if err != nil {
			errors = multierror.Append(errors, err)
			continue
		}
		return id, nil
	}
	return "", MakeInternal(fmt.Errorf("cannot get sdk id: %w", errors))
}

// GetSdkOwnerID returns sdk ownerID from sdk identity certificate.
func (c *Client) GetSdkOwnerID() (string, error) {
	id, err := getSdkOwnerID(c.tlsConfig.GetCertificate)
	if err != nil {
		return "", getSdkError(err)
	}
	return id, nil
}

// GetSdkOwnerID returns sdk ownerID
func (d *Device) GetSdkOwnerID() (string, error) {
	if d.cfg.GetOwnerID != nil {
		return d.cfg.GetOwnerID()
	}

	id, err := getSdkOwnerID(d.cfg.TLSConfig.GetCertificate)
	if err != nil {
		return "", getSdkError(err)
	}
	return id, nil
}
