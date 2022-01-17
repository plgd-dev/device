package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/plgd-dev/device/client/core"
	pkgError "github.com/plgd-dev/device/pkg/error"
)

type deviceOwnershipNone struct {
}

func NewDeviceOwnershipNone() *deviceOwnershipNone {
	return &deviceOwnershipNone{}
}

type noneSigner struct {
}

func (s noneSigner) Sign(context.Context, []byte) ([]byte, error) {
	return nil, fmt.Errorf("sign is not supported by %T", s)
}

func (o *deviceOwnershipNone) GetIdentitySigner(accessToken string) core.CertificateSigner {
	return noneSigner{}
}

func (o *deviceOwnershipNone) OwnDevice(ctx context.Context, deviceID string, otmType OTMType, discoveryConfiguration core.DiscoveryConfiguration, own ownFunc, opts ...core.OwnOption) (string, error) {
	return own(ctx, deviceID, nil, discoveryConfiguration, opts...)
}

func (o *deviceOwnershipNone) Initialization(ctx context.Context) error {
	return nil
}

func (o *deviceOwnershipNone) GetIdentityCertificate() (tls.Certificate, error) {
	return tls.Certificate{}, pkgError.NotSupported()
}

func (o *deviceOwnershipNone) GetAccessTokenURL(ctx context.Context) (string, error) {
	return "", pkgError.NotSupported()
}

func (o *deviceOwnershipNone) GetOnboardAuthorizationCodeURL(ctx context.Context, deviceID string) (string, error) {
	return "", pkgError.NotSupported()
}

func (o *deviceOwnershipNone) GetIdentityCACerts() ([]*x509.Certificate, error) {
	return nil, pkgError.NotSupported()
}
