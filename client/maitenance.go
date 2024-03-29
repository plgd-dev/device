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

	"github.com/plgd-dev/device/v2/pkg/net/coap"
)

// FactoryReset factory resets the device.
func (c *Client) FactoryReset(ctx context.Context, deviceID string, opts ...CommonCommandOption) error {
	cfg := applyCommonOptions(opts...)
	d, links, err := c.GetDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}
	if c.useDeviceIDInQuery {
		cfg.opts = append(cfg.opts, coap.WithDeviceID(deviceID))
	}

	defer c.removeTemporaryDeviceFromCache(ctx, d)
	return d.FactoryReset(ctx, links, cfg.opts...)
}

// Reboot reboots the device.
func (c *Client) Reboot(ctx context.Context, deviceID string, opts ...CommonCommandOption) error {
	cfg := applyCommonOptions(opts...)
	d, links, err := c.GetDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}
	if c.useDeviceIDInQuery {
		cfg.opts = append(cfg.opts, coap.WithDeviceID(deviceID))
	}

	defer c.removeTemporaryDeviceFromCache(ctx, d)
	return d.Reboot(ctx, links, cfg.opts...)
}
