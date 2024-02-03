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

package service

import (
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/pkg/log"
)

type OptionsCfg struct {
	onDiscoveryDevices func(req *net.Request)
	logger             log.Logger
}

func WithOnDiscoveryDevices(f func(req *net.Request)) Option {
	return func(o *OptionsCfg) {
		if f != nil {
			o.onDiscoveryDevices = f
		}
	}
}

func WithLogger(logger log.Logger) Option {
	return func(o *OptionsCfg) {
		o.logger = logger
	}
}

type Option func(*OptionsCfg)
