package local

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/plgd-dev/sdk/local/core"
	"github.com/plgd-dev/sdk/local/core/otm/manufacturer"

	"github.com/plgd-dev/kit/security"
	ocfSigner "github.com/plgd-dev/kit/security/signer"
	"github.com/google/uuid"
)

type DeviceOwnershipSDKConfig struct {
	ID         string
	Cert       []byte
	CertKey    []byte
	CertExpiry *string
}

type deviceOwnershipSDK struct {
	sdkDeviceID         string
	identitySigner      core.CertificateSigner
	identityCertificate tls.Certificate
	disableDTLS         bool
	app                 ApplicationCallback
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
	return NewDeviceOwnershipSDK(app, uid.String(), &signerCert, certExpiry, disableDTLS)
}

func NewDeviceOwnershipSDK(app ApplicationCallback, sdkDeviceID string, signerCert *tls.Certificate, certExpiry time.Duration, disableDTLS bool) (*deviceOwnershipSDK, error) {
	signerCAs, err := security.ParseX509Certificates(signerCert)
	if err != nil {
		return nil, fmt.Errorf("could not parse signer certificates")
	}
	return &deviceOwnershipSDK{
		sdkDeviceID:    sdkDeviceID,
		identitySigner: ocfSigner.NewIdentityCertificateSigner(signerCAs, signerCert.PrivateKey, certExpiry),
		disableDTLS:    disableDTLS,
		app:            app,
	}, nil
}

func (o *deviceOwnershipSDK) Close(ctx context.Context) error {
	return nil
}

func getOTMManufacturer(app ApplicationCallback, disableDTLS bool, signer core.CertificateSigner) (core.OTMClient, error) {
	certAuthorities, err := app.GetRootCertificateAuthorities()
	if err != nil {
		return nil, err
	}
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

	return manufacturer.NewClient(mfgCert, mfgCA, signer, certAuthorities, mfgOpts...), nil
}

func (o *deviceOwnershipSDK) OwnDevice(ctx context.Context, deviceID string, own ownFunc, opts ...core.OwnOption) error {
	otm, err := getOTMManufacturer(o.app, o.disableDTLS, o.identitySigner)
	if err != nil {
		return err
	}
	return own(ctx, deviceID, otm, opts...)
}

func (o *deviceOwnershipSDK) Initialization(ctx context.Context) error {
	cert, err := GenerateSDKIdentityCertificate(ctx, o.identitySigner, o.sdkDeviceID)
	o.identityCertificate = cert
	return err
}

func (o *deviceOwnershipSDK) GetIdentityCertificate() (tls.Certificate, error) {
	if o.identityCertificate.PrivateKey == nil {
		return tls.Certificate{}, fmt.Errorf("client is not initialized")
	}
	return o.identityCertificate, nil
}

func (o *deviceOwnershipSDK) GetAccessTokenURL(ctx context.Context) (string, error) {
	return "", fmt.Errorf("not supported")
}

func (o *deviceOwnershipSDK) GetOnboardAuthorizationCodeURL(ctx context.Context, deviceID string) (string, error) {
	return "", fmt.Errorf("not supported")
}
