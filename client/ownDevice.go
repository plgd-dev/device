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

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/client/core/otm"
)

// OwnDevice transfer ownership to the client and setup time at the device.
// In the absence of a cached device, it is found through multicast and stored with an expiration time.
func (c *Client) OwnDevice(ctx context.Context, deviceID string, opts ...OwnOption) (string, error) {
	cfg := ownOptions{
		otmTypes:               []OTMType{OTMType_JustWorks},
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnOwn(cfg)
	}
	d, _, err := c.GetDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return "", err
	}
	ok := d.IsSecured()
	if !ok {
		// don't own insecure device
		return deviceID, nil
	}
	return c.deviceOwner.OwnDevice(ctx, deviceID, cfg.otmTypes, cfg.discoveryConfiguration, c.ownDeviceWithSigners, cfg.opts...)
}

func (c *Client) updateCache(d *core.Device, oldDeviceID string) {
	if d.DeviceID() == oldDeviceID {
		return
	}
	// we need to move device in cache because deviceID is changed: from oldDeviceID to d.DeviceID()
	// store the device with new deviceID key
	exp, ok := c.deviceCache.GetDeviceExpiration(oldDeviceID)
	if ok && exp.IsZero() {
		c.deviceCache.UpdateOrStoreDevice(d)
	} else {
		c.deviceCache.UpdateOrStoreDeviceWithExpiration(d)
	}
	// remove device from key oldDeviceID
	// we don't need to close it because it is already stored on new deviceID position
	_, _ = c.deviceCache.LoadAndDeleteDevice(oldDeviceID)
}

func (c *Client) ownDeviceWithSigners(ctx context.Context, deviceID string, otmClient []otm.Client, discoveryConfiguration core.DiscoveryConfiguration, opts ...core.OwnOption) (string, error) {
	d, links, err := c.GetDevice(ctx, deviceID, WithDiscoveryConfiguration(discoveryConfiguration))
	if err != nil {
		return "", err
	}
	ok := d.IsSecured()
	if !ok {
		// don't own insecure device
		return d.DeviceID(), nil
	}
	if c.disableUDPEndpoints {
		// we need to get all links because just-works need to use dtls
		endpoints := d.GetEndpoints()
		links, err = d.GetResourceLinks(ctx, endpoints)
		if err != nil {
			return "", err
		}
		links = patchResourceLinksEndpoints(links, false)
	}

	err = d.Own(ctx, links, otmClient, opts...)
	if err != nil {
		return "", err
	}
	c.updateCache(d, deviceID)

	return d.DeviceID(), nil
}
