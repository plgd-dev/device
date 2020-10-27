package local

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/plgd-dev/sdk/local/core"
	justworks "github.com/plgd-dev/sdk/local/core/otm/just-works"
	"github.com/plgd-dev/sdk/local/core/otm/manufacturer"

	"github.com/google/uuid"
	"github.com/karrick/tparse/v2"
	"github.com/plgd-dev/kit/security"
	ocfSigner "github.com/plgd-dev/kit/security/signer"
)

type DeviceOwnershipSDKConfig struct {
	ID         string
	Cert       []byte
	CertKey    []byte
	ValidFrom  string //RFC3339, or now-1m, empty means now-1m
	CertExpiry *string
}

type deviceOwnershipSDK struct {
	sdkDeviceID          string
	createIdentitySigner func() (core.CertificateSigner, error)
	identityCertificate  tls.Certificate
	identityCACert       []*x509.Certificate
	disableDTLS          bool
	app                  ApplicationCallback
}

func NewDeviceOwnershipSDKFromConfig(app ApplicationCallback, cfg *DeviceOwnershipSDKConfig, disableDTLS bool) (*deviceOwnershipSDK, error) {
	certExpiry := time.Hour * 24 * 365 * 10
	var err error
	if cfg.CertExpiry != nil {
		certExpiry, err = time.ParseDuration(*cfg.CertExpiry)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid cert expiry for device ownership SDK: %w", err)
	}
	signerCert, err := tls.X509KeyPair(cfg.Cert, cfg.CertKey)
	if err != nil {
		return nil, fmt.Errorf("invalid cert or key for device ownership SDK: %w", err)
	}
	uid, err := uuid.Parse(cfg.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid ID for device ownership SDK: %w", err)
	}

	return NewDeviceOwnershipSDK(app, uid.String(), &signerCert, cfg.ValidFrom, certExpiry, disableDTLS)
}

func NewDeviceOwnershipSDK(app ApplicationCallback, sdkDeviceID string, signerCert *tls.Certificate, validFrom string, certExpiry time.Duration, disableDTLS bool) (*deviceOwnershipSDK, error) {
	if validFrom == "" {
		validFrom = "now-1m"
	}
	_, err := tparse.ParseNow(time.RFC3339, validFrom)
	if err != nil {
		return nil, fmt.Errorf("invalid validFrom(%v) for device ownership SDK: %w", validFrom, err)
	}

	signerCAs, err := security.ParseX509Certificates(signerCert)
	if err != nil {
		return nil, fmt.Errorf("could not parse signer certificates")
	}
	return &deviceOwnershipSDK{
		sdkDeviceID: sdkDeviceID,
		createIdentitySigner: func() (core.CertificateSigner, error) {
			notBefore, err := tparse.ParseNow(time.RFC3339, validFrom)
			if err != nil {
				return nil, fmt.Errorf("invalid validFrom(%v): %w", validFrom, err)
			}
			notAfter := notBefore.Add(certExpiry)
			return ocfSigner.NewIdentityCertificateSigner(signerCAs, signerCert.PrivateKey, notBefore, notAfter), nil
		},
		disableDTLS: disableDTLS,
		app:         app,
	}, nil
}

func (o *deviceOwnershipSDK) Close(ctx context.Context) error {
	return nil
}

func getOTMManufacturer(app ApplicationCallback, disableDTLS bool, signer core.CertificateSigner) (core.OTMClient, error) {
	mfgCA, err := app.GetManufacturerCertificateAuthorities()
	if err != nil {
		return nil, err
	}
	mfgCert, err := app.GetManufacturerCertificate()
	if err != nil {
		return nil, err
	}

	mfgOpts := make([]manufacturer.OptionFunc, 0, 1)
	if disableDTLS {
		mfgOpts = append(mfgOpts, manufacturer.WithoutDTLS())
	}

	return manufacturer.NewClient(mfgCert, mfgCA, signer, mfgOpts...), nil
}

func (o *deviceOwnershipSDK) OwnDevice(ctx context.Context, deviceID string, otmType OTMType, own ownFunc, opts ...core.OwnOption) (string, error) {
	signer, err := o.createIdentitySigner()
	if err != nil {
		return "", err
	}
	var otmClient core.OTMClient
	switch otmType {
	case OTMType_Manufacturer:
		otm, err := getOTMManufacturer(o.app, o.disableDTLS, signer)
		if err != nil {
			return "", err
		}
		otmClient = otm
	case OTMType_JustWorks:
		otmClient = justworks.NewClient(signer)
	default:
		return "", fmt.Errorf("unsupported ownership transfer method: %v", otmType)
	}
	return own(ctx, deviceID, otmClient, opts...)
}

func (o *deviceOwnershipSDK) Initialization(ctx context.Context) error {
	signer, err := o.createIdentitySigner()
	if err != nil {
		return err
	}
	cert, caCert, err := GenerateSDKIdentityCertificate(ctx, signer, o.sdkDeviceID)
	o.identityCertificate = cert
	o.identityCACert = caCert
	return err
}

func (o *deviceOwnershipSDK) GetIdentityCertificate() (tls.Certificate, error) {
	if o.identityCertificate.PrivateKey == nil {
		return tls.Certificate{}, fmt.Errorf("client is not initialized")
	}
	return o.identityCertificate, nil
}

func (o *deviceOwnershipSDK) GetIdentityCACerts() ([]*x509.Certificate, error) {
	if o.identityCACert == nil {
		return nil, fmt.Errorf("client is not initialized")
	}
	return o.identityCACert, nil
}

func (o *deviceOwnershipSDK) GetAccessTokenURL(ctx context.Context) (string, error) {
	return "", fmt.Errorf("not supported")
}

func (o *deviceOwnershipSDK) GetOnboardAuthorizationCodeURL(ctx context.Context, deviceID string) (string, error) {
	return "", fmt.Errorf("not supported")
}
