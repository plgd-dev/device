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
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/plgd-dev/device/v2/client/core"
	pkgError "github.com/plgd-dev/device/v2/pkg/error"
)

type deviceOwnershipNone struct{}

func NewDeviceOwnershipNone() *deviceOwnershipNone {
	return &deviceOwnershipNone{}
}

type noneSigner struct{}

func (s noneSigner) Sign(context.Context, []byte) ([]byte, error) {
	return nil, fmt.Errorf("sign is not supported by %T", s)
}

func (o *deviceOwnershipNone) GetIdentitySigner(accessToken string) core.CertificateSigner {
	return noneSigner{}
}

func (o *deviceOwnershipNone) OwnDevice(ctx context.Context, deviceID string, otmTypes []OTMType, discoveryConfiguration core.DiscoveryConfiguration, own ownFunc, opts ...core.OwnOption) (string, error) {
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
