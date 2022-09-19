package core

import (
	"crypto/x509"
	"fmt"

	"github.com/plgd-dev/device/pkg/net/coap"
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

	var errors []error

	for _, c := range cert.Certificate {
		x509cert, err := x509.ParseCertificate(c)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		id, err := coap.GetDeviceIDFromIdentityCertificate(x509cert)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		return id, nil
	}
	return "", MakeInternal(fmt.Errorf("cannot get sdk id: %v", errors))
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
