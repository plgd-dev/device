/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package device

import (
	"crypto/x509"

	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/resources/device"
	"github.com/plgd-dev/device/v2/pkg/log"
)

type OnDeviceUpdated func(d *Device)

type CAPoolGetter interface {
	IsValid() bool
	GetPool() (*x509.CertPool, error)
}

type OptionsCfg struct {
	onDeviceUpdated         OnDeviceUpdated
	getAdditionalProperties device.GetAdditionalPropertiesForResponseFunc
	getCertificates         cloud.GetCertificates
	caPool                  CAPoolGetter
	logger                  log.Logger
}

type Option func(*OptionsCfg)

func WithOnDeviceUpdated(onDeviceUpdated OnDeviceUpdated) Option {
	return func(o *OptionsCfg) {
		o.onDeviceUpdated = onDeviceUpdated
	}
}

func WithGetAdditionalPropertiesForResponse(getAdditionalProperties device.GetAdditionalPropertiesForResponseFunc) Option {
	return func(o *OptionsCfg) {
		o.getAdditionalProperties = getAdditionalProperties
	}
}

func WithGetCertificates(getCertificates cloud.GetCertificates) Option {
	return func(o *OptionsCfg) {
		o.getCertificates = getCertificates
	}
}

func WithCAPool(caPool CAPoolGetter) Option {
	return func(o *OptionsCfg) {
		o.caPool = caPool
	}
}

func WithLogger(logger log.Logger) Option {
	return func(o *OptionsCfg) {
		o.logger = logger
	}
}
