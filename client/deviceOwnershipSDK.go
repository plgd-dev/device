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

package client

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/client/core/otm"
	justworks "github.com/plgd-dev/device/v2/client/core/otm/just-works"
	"github.com/plgd-dev/device/v2/client/core/otm/manufacturer"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
)

type Signer = interface {
	Sign()
}

type DeviceOwnershipSDKConfig struct {
	ID         string
	Cert       string
	CertKey    string
	ValidFrom  string // RFC3339, or now-1m, empty means now-1m
	CertExpiry *string

	CreateSignerFunc func(caCert []*x509.Certificate, caKey crypto.PrivateKey, validNotBefore time.Time, validNotAfter time.Time) core.CertificateSigner
}

type deviceOwnershipSDK struct {
	sdkDeviceID          string
	createIdentitySigner func() (core.CertificateSigner, error)
	identityCertificate  tls.Certificate
	identityCACert       []*x509.Certificate
	dialTLS              core.DialTLS
	dialDTLS             core.DialDTLS
	app                  ApplicationCallback
}

func newDeviceOwnershipSDKFromConfig(app ApplicationCallback, dialTLS core.DialTLS,
	dialDLTS core.DialDTLS, cfg *DeviceOwnershipSDKConfig,
) (*deviceOwnershipSDK, error) {
	certExpiry := time.Hour * 24 * 365 * 10
	var err error
	if cfg.CertExpiry != nil {
		certExpiry, err = time.ParseDuration(*cfg.CertExpiry)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid cert expiry for device ownership SDK: %w", err)
	}
	signerCert, err := tls.X509KeyPair([]byte(cfg.Cert), []byte(cfg.CertKey))
	if err != nil {
		return nil, fmt.Errorf("invalid cert or key for device ownership SDK: %w", err)
	}
	uid, err := uuid.Parse(cfg.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid ID for device ownership SDK: %w", err)
	}

	return newDeviceOwnershipSDK(app, uid.String(), dialTLS, dialDLTS, &signerCert, cfg.ValidFrom, certExpiry, cfg.CreateSignerFunc)
}

func newDeviceOwnershipSDK(app ApplicationCallback, sdkDeviceID string, dialTLS core.DialTLS,
	dialDTLS core.DialDTLS, signerCert *tls.Certificate, validFrom string, certExpiry time.Duration, createSigner func(caCert []*x509.Certificate, caKey crypto.PrivateKey, validNotBefore time.Time, validNotAfter time.Time) core.CertificateSigner,
) (*deviceOwnershipSDK, error) {
	if validFrom == "" {
		validFrom = "now-1m"
	}
	_, err := tparse.ParseNow(time.RFC3339, validFrom)
	if err != nil {
		return nil, fmt.Errorf("invalid validFrom(%v) for device ownership SDK: %w", validFrom, err)
	}

	signerCAs, err := pkgX509.ParseCertificates(signerCert)
	if err != nil {
		return nil, errors.New("could not parse signer certificates")
	}
	return &deviceOwnershipSDK{
		sdkDeviceID: sdkDeviceID,
		createIdentitySigner: func() (core.CertificateSigner, error) {
			if createSigner == nil {
				return nil, errors.New("create signer is not set")
			}
			notBefore, err := tparse.ParseNow(time.RFC3339, validFrom)
			if err != nil {
				return nil, fmt.Errorf("invalid validFrom(%v): %w", validFrom, err)
			}
			notAfter := notBefore.Add(certExpiry)
			return createSigner(signerCAs, signerCert.PrivateKey, notBefore, notAfter), nil
		},
		app:      app,
		dialTLS:  dialTLS,
		dialDTLS: dialDTLS,
	}, nil
}

func getOTMManufacturer(app ApplicationCallback, dialTLS core.DialTLS,
	dialDTLS core.DialDTLS,
) (otm.Client, error) {
	mfgCA, err := app.GetManufacturerCertificateAuthorities()
	if err != nil {
		return nil, err
	}
	mfgCert, err := app.GetManufacturerCertificate()
	if err != nil {
		return nil, err
	}

	return manufacturer.NewClient(mfgCert, mfgCA, manufacturer.WithDialDTLS(dialDTLS), manufacturer.WithDialTLS(dialTLS)), nil
}

func getOtmClients(app ApplicationCallback, dialTLS core.DialTLS, dialDTLS core.DialDTLS, otmTypes []OTMType) ([]otm.Client, error) {
	otmClients := make([]otm.Client, 0, 2)
	for _, otmType := range otmTypes {
		switch otmType {
		case OTMType_Manufacturer:
			otm, err := getOTMManufacturer(app, dialTLS, dialDTLS)
			if err != nil {
				return nil, err
			}
			otmClients = append(otmClients, otm)
		case OTMType_JustWorks:
			otmClients = append(otmClients, justworks.NewClient(justworks.WithDialDTLS(dialDTLS)))
		default:
			return nil, fmt.Errorf("unsupported ownership transfer method: %v", otmType)
		}
	}
	return otmClients, nil
}

func (o *deviceOwnershipSDK) OwnDevice(ctx context.Context, deviceID string, otmTypes []OTMType, discoveryConfiguration core.DiscoveryConfiguration, own ownFunc, opts ...core.OwnOption) (string, error) {
	signer, err := o.createIdentitySigner()
	if err != nil {
		return "", err
	}
	otmClients, err := getOtmClients(o.app, o.dialTLS, o.dialDTLS, otmTypes)
	if err != nil {
		return "", err
	}
	opts = append([]core.OwnOption{core.WithSetupCertificates(signer.Sign)}, opts...)
	return own(ctx, deviceID, otmClients, discoveryConfiguration, opts...)
}

func (o *deviceOwnershipSDK) Initialization(ctx context.Context) error {
	signer, err := o.createIdentitySigner()
	if err != nil {
		return err
	}
	cert, caCert, err := GenerateSDKIdentityCertificate(ctx, signer.Sign, o.sdkDeviceID)
	o.identityCertificate = cert
	o.identityCACert = caCert
	return err
}

func (o *deviceOwnershipSDK) GetIdentityCertificate() (tls.Certificate, error) {
	if o.identityCertificate.PrivateKey == nil {
		return tls.Certificate{}, errors.New("client is not initialized")
	}
	return o.identityCertificate, nil
}

func (o *deviceOwnershipSDK) GetIdentityCACerts() ([]*x509.Certificate, error) {
	if o.identityCACert == nil {
		return nil, errors.New("client is not initialized")
	}
	return o.identityCACert, nil
}
