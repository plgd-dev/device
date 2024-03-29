/****************************************************************************
 *
 * Copyright (c) 2023 plgd.dev s.r.o.
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

package cloud

import (
	"time"

	"github.com/plgd-dev/device/v2/pkg/log"
)

type OptionsCfg struct {
	maxMessageSize  uint32
	getCertificates GetCertificates
	removeCloudCAs  RemoveCloudCAs
	logger          log.Logger
	tickInterval    time.Duration
}

type Option func(*OptionsCfg)

func WithMaxMessageSize(maxMessageSize uint32) Option {
	return func(o *OptionsCfg) {
		if maxMessageSize > 0 {
			o.maxMessageSize = maxMessageSize
		}
	}
}

func WithGetCertificates(getCertificates GetCertificates) Option {
	return func(o *OptionsCfg) {
		o.getCertificates = getCertificates
	}
}

func WithRemoveCloudCAs(removeCloudCA RemoveCloudCAs) Option {
	return func(o *OptionsCfg) {
		o.removeCloudCAs = removeCloudCA
	}
}

func WithLogger(logger log.Logger) Option {
	return func(o *OptionsCfg) {
		o.logger = logger
	}
}

func WithTickInterval(t time.Duration) Option {
	return func(o *OptionsCfg) {
		o.tickInterval = t
	}
}
