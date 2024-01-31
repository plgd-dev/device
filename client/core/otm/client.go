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

package otm

import (
	"context"
	"encoding/pem"
	"fmt"

	"github.com/plgd-dev/device/v2/pkg/net/coap"
	pkgX509 "github.com/plgd-dev/device/v2/pkg/security/x509"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/schema/csr"
	"github.com/plgd-dev/device/v2/schema/doxm"
	kitNet "github.com/plgd-dev/kit/v2/net"
)

// SignFunc handles a certifice signing request (csr), the csr and returned certificate chain are encoded in PEM format
type SignFunc = func(ctx context.Context, csr []byte) ([]byte, error)

type Client interface {
	Type() doxm.OwnerTransferMethod
	Dial(ctx context.Context, addr kitNet.Addr) (*coap.ClientCloseHandler, error)
}

func encodeToPem(encoding csr.CertificateEncoding, data []byte) []byte {
	if encoding == csr.CertificateEncoding_DER {
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: data})
	}
	return data
}

type SetupCertificatesOption struct {
	sign     SignFunc
	deviceID string
}

type provisionOwnerCredentialstOptions struct {
	setupCertificates *SetupCertificatesOption
}

func (o SetupCertificatesOption) applyProvisionOwnerCredentials(opts provisionOwnerCredentialstOptions) provisionOwnerCredentialstOptions {
	opts.setupCertificates = &o
	return opts
}

type ProvisionOwnerCredentialstOption interface {
	applyProvisionOwnerCredentials(opts provisionOwnerCredentialstOptions) provisionOwnerCredentialstOptions
}

func WithSetupCertificates(deviceID string, sign SignFunc) SetupCertificatesOption {
	return SetupCertificatesOption{
		sign:     sign,
		deviceID: deviceID,
	}
}

func validateProvisionOwnerCredentials(ownerID string, psk []byte, opts provisionOwnerCredentialstOptions) error {
	if ownerID == "" {
		return fmt.Errorf("invalid ownerID")
	}
	if opts.setupCertificates != nil {
		if opts.setupCertificates.deviceID == "" {
			return fmt.Errorf("invalid deviceID")
		}
		if opts.setupCertificates.sign == nil {
			return fmt.Errorf("invalid sign")
		}
	}
	if len(psk) == 0 {
		return fmt.Errorf("invalid preshared key")
	}
	if len(psk) != 16 {
		return fmt.Errorf("size of preshared key('%v') must be 16bytes", len(psk))
	}
	return nil
}

func ProvisionOwnerCredentials(ctx context.Context, tlsClient *coap.ClientCloseHandler, ownerID string, psk []byte, opts ...ProvisionOwnerCredentialstOption) error {
	var cfg provisionOwnerCredentialstOptions
	for _, o := range opts {
		cfg = o.applyProvisionOwnerCredentials(cfg)
	}
	if err := validateProvisionOwnerCredentials(ownerID, psk, cfg); err != nil {
		return err
	}

	/*setup credentials - PostOwnerCredential*/
	setDeviceCredentials := credential.CredentialUpdateRequest{
		ResourceOwner: ownerID,
		Credentials: []credential.Credential{
			{
				Subject: ownerID,
				Type:    credential.CredentialType_SYMMETRIC_PAIR_WISE,
				PrivateData: &credential.CredentialPrivateData{
					DataInternal: string(psk),
					Encoding:     credential.CredentialPrivateDataEncoding_RAW,
				},
			},
		},
	}
	if cfg.setupCertificates != nil {
		var r csr.CertificateSigningRequestResponse
		err := tlsClient.GetResource(ctx, csr.ResourceURI, &r)
		if err != nil {
			return fmt.Errorf("cannot get csr for setup device owner credentials: %w", err)
		}

		pemCSR := encodeToPem(r.Encoding, r.CSR())

		signedCsr, err := cfg.setupCertificates.sign(ctx, pemCSR)
		if err != nil {
			return fmt.Errorf("cannot sign csr for setup device owner credentials: %w", err)
		}

		certsFromChain, err := pkgX509.ParsePemCertificates(signedCsr)
		if err != nil {
			return fmt.Errorf("failed to parse chain of X509 certs: %w", err)
		}
		setDeviceCredentials.Credentials = append(setDeviceCredentials.Credentials,
			credential.Credential{
				Subject: cfg.setupCertificates.deviceID,
				Type:    credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
				Usage:   credential.CredentialUsage_CERT,
				PublicData: &credential.CredentialPublicData{
					DataInternal: string(signedCsr),
					Encoding:     credential.CredentialPublicDataEncoding_PEM,
				},
			},
			credential.Credential{
				Subject: ownerID,
				Type:    credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
				Usage:   credential.CredentialUsage_TRUST_CA,
				PublicData: &credential.CredentialPublicData{
					DataInternal: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certsFromChain[len(certsFromChain)-1].Raw})),
					Encoding:     credential.CredentialPublicDataEncoding_PEM,
				},
			})
	}
	err := tlsClient.UpdateResource(ctx, credential.ResourceURI, setDeviceCredentials, nil)
	if err != nil {
		return fmt.Errorf("cannot set device credentials: %w", err)
	}

	return nil
}
