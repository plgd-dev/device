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

	"github.com/plgd-dev/device/v2/client/core"
	pkgError "github.com/plgd-dev/device/v2/pkg/error"
)

type deviceOwnershipNone struct{}

func newDeviceOwnershipNone() *deviceOwnershipNone {
	return &deviceOwnershipNone{}
}

func (o *deviceOwnershipNone) Initialization(context.Context) error {
	return nil
}

func (o *deviceOwnershipNone) OwnDevice(ctx context.Context, deviceID string, _ []OTMType, discoveryConfiguration core.DiscoveryConfiguration, own ownFunc, opts ...core.OwnOption) (string, error) {
	return own(ctx, deviceID, nil, discoveryConfiguration, opts...)
}

func (o *deviceOwnershipNone) GetIdentityCertificate() (tls.Certificate, error) {
	return tls.Certificate{}, pkgError.NotSupported()
}

func (o *deviceOwnershipNone) GetIdentityCACerts() ([]*x509.Certificate, error) {
	return nil, pkgError.NotSupported()
}
