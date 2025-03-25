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
	"crypto/x509"
	"errors"
	"fmt"
	"strings"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/status"
)

func getLinksDevice(ctx context.Context, dev *core.Device, disableUDPEndpoints bool) (schema.ResourceLinks, error) {
	endpoints := dev.GetEndpoints()
	links, err := dev.GetResourceLinks(ctx, endpoints)
	if err != nil {
		return nil, err
	}
	return patchResourceLinksEndpoints(links, disableUDPEndpoints), nil
}

// Don't remove devices found by IP, the device is probably offline
// and we will be not able to reestablish the connection when it will
// come back online
func (c *Client) deleteDeviceNotFoundByIP(ctx context.Context, dev *core.Device) {
	if dev.FoundByIP() == "" {
		c.deviceCache.LoadAndDeleteDevice(dev.DeviceID())
	}
	if err := dev.Close(ctx); err != nil {
		c.logger.Debugf("delete device error: %s", err.Error())
	}
}

// GetDeviceByMulticast gets device by multicast and store it to cache with expiration.
// When the device expiration time has expired, the device will be removed from cache.
// The device expiration time is prolonged by using the device.
func (c *Client) GetDeviceByMulticast(ctx context.Context, deviceID string, opts ...GetDeviceOption) (*core.Device, schema.ResourceLinks, error) {
	cfg := getDeviceOptions{
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnGetDevice(cfg)
	}
	dev, err := c.client.GetDeviceByMulticast(ctx, deviceID, cfg.discoveryConfiguration)
	if err != nil {
		return nil, nil, err
	}
	links, err := getLinksDevice(ctx, dev, c.disableUDPEndpoints)
	if err != nil {
		if errC := dev.Close(ctx); errC != nil {
			c.logger.Debugf("get links for device error: %w", errC)
		}
		return nil, nil, fmt.Errorf("cannot get links for device %v: %w", deviceID, err)
	}
	retDev, updated := c.deviceCache.UpdateOrStoreDeviceWithExpiration(dev)
	if updated {
		if errC := dev.Close(ctx); errC != nil {
			c.logger.Debugf("get device by multicast error: %w", errC)
		}
	}
	return retDev, links, nil
}

func (c *Client) getDeviceByIP(ctx context.Context, ip string, expectedDeviceID string) (*core.Device, schema.ResourceLinks, error) {
	dev, err := c.getDeviceByIPWithUpdateCache(ctx, strings.Trim(ip, "[]"), expectedDeviceID)
	if err != nil {
		return nil, nil, err
	}
	links, err := getLinksDevice(ctx, dev, c.disableUDPEndpoints)
	if err != nil {
		return nil, nil, err
	}
	return dev, links, nil
}

func (c *Client) getDeviceByIPWithUpdateCache(ctx context.Context, ip string, expectedDeviceID string) (*core.Device, error) {
	newDev, err := c.client.GetDeviceByIP(ctx, ip)
	if err != nil {
		return nil, err
	}
	var oldDev *core.Device
	if expectedDeviceID != "" {
		oldDev, _ = c.deviceCache.GetDevice(expectedDeviceID)
	} else {
		oldDev = c.deviceCache.GetDeviceByFoundIP(ip)
	}
	if oldDev != nil && oldDev.DeviceID() != newDev.DeviceID() {
		tmp, ok := c.deviceCache.LoadAndDeleteDevice(oldDev.DeviceID())
		if ok && tmp == oldDev {
			oldDev.UpdateBy(newDev)
			if errC := newDev.Close(ctx); errC != nil {
				c.logger.Debugf("get device by ip error: %w", errC)
			}
			newDev = oldDev
		}
	}
	dev, _ := c.deviceCache.UpdateOrStoreDevice(newDev)
	return dev, nil
}

func (c *Client) checkAndUpdateCacheByLinks(ctx context.Context, dev *core.Device, links schema.ResourceLinks) (*core.Device, schema.ResourceLinks, error) {
	devLinks := links.GetResourceLinks(device.ResourceType)
	if len(devLinks) == 0 {
		return nil, nil, fmt.Errorf("cannot get %v resourceType for device %v: not found", device.ResourceType, dev.DeviceID())
	}
	if devLinks[0].GetDeviceID() == dev.DeviceID() {
		return dev, links, nil
	}
	newDeviceID := devLinks[0].GetDeviceID()
	tmp, ok := c.deviceCache.LoadAndDeleteDevice(dev.DeviceID())
	if ok {
		tmp.SetDeviceID(newDeviceID)
		_, updated := c.deviceCache.UpdateOrStoreDeviceWithExpiration(tmp)
		if updated {
			if errC := tmp.Close(ctx); errC != nil {
				c.logger.Debugf("update device cache error: %w", errC)
			}
		}
	}
	return nil, nil, fmt.Errorf("cannot get device %v: not found", dev.DeviceID())
}

// GetDevice gets the device from the cache or via multicast or via IP address if was previously stored by GetDeviceByIP and updates device in the cache.
func (c *Client) GetDevice(ctx context.Context, deviceID string, opts ...GetDeviceOption,
) (*core.Device, schema.ResourceLinks, error) {
	dev, ok := c.deviceCache.GetDevice(deviceID)
	if !ok {
		return c.GetDeviceByMulticast(ctx, deviceID, opts...)
	}
	links, err := getLinksDevice(ctx, dev, c.disableUDPEndpoints)
	if err == nil {
		return c.checkAndUpdateCacheByLinks(ctx, dev, links)
	}
	var newDev *core.Device
	if dev.FoundByIP() != "" {
		newDev, links, err = c.getDeviceByIP(ctx, dev.FoundByIP(), deviceID)
		if err != nil {
			return nil, nil, err
		}
		if newDev.DeviceID() != deviceID {
			return nil, nil, fmt.Errorf("cannot get device %v: not found", deviceID)
		}
		return dev, links, nil
	}
	c.deleteDeviceNotFoundByIP(ctx, dev)
	return c.GetDevice(ctx, deviceID, opts...)
}

// GetDeviceByIP gets device by IP and store it to cache without expiration.
// To delete device, call DeleteDevices with the deviceID.
func (c *Client) GetDeviceByIP(
	ctx context.Context,
	ip string,
) (*core.Device, schema.ResourceLinks, error) {
	return c.getDeviceByIP(ctx, ip, "")
}

func isDeviceOwnedByOther(err error) bool {
	if v, ok := status.FromError(err); ok && v.Code() == codes.Unauthorized {
		return true
	}
	var unknownAuth x509.UnknownAuthorityError
	return errors.As(err, &unknownAuth)
}

func (c *Client) getDeviceDetails(ctx context.Context, dev *core.Device, links schema.ResourceLinks, getDetails GetDetailsFunc) (DeviceDetails, error) {
	devDetails, err := getDeviceDetails(ctx, dev, links, getDetails)
	if err != nil {
		return DeviceDetails{}, err
	}
	var o ownership
	if devDetails.IsSecured {
		d, ownErr := dev.GetOwnership(ctx, links)
		if ownErr != nil {
			if isDeviceOwnedByOther(ownErr) {
				o.status = OwnershipStatus_OwnedByOther
			}
		} else {
			o.doxm = &d
		}
	}
	ownerID, _ := c.client.GetSdkOwnerID()

	return setOwnership(ownerID, map[string]DeviceDetails{
		devDetails.ID: devDetails,
	}, map[string]ownership{
		devDetails.ID: o,
	})[devDetails.ID], nil
}

func (c *Client) GetDeviceDetailsByMulticast(ctx context.Context, deviceID string, opts ...GetDeviceOption) (DeviceDetails, error) {
	cfg := getDeviceOptions{
		getDetails:             getDetails,
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnGetDevice(cfg)
	}

	dev, links, err := c.GetDevice(ctx, deviceID, opts...)
	if err != nil {
		return DeviceDetails{}, err
	}
	return c.getDeviceDetails(ctx, dev, links, cfg.getDetails)
}

func (c *Client) GetAllDeviceIDsFoundByIP() map[string]string {
	return c.deviceCache.GetDevicesFoundByIP()
}

// GetDeviceDetailsByIP gets the device directly via IP address and multicast listen port 5683.
func (c *Client) GetDeviceDetailsByIP(ctx context.Context, ip string, opts ...GetDeviceByIPOption) (DeviceDetails, error) {
	cfg := getDeviceByIPOptions{
		getDetails: getDetails,
	}
	for _, o := range opts {
		cfg = o.applyOnGetDeviceByIP(cfg)
	}

	dev, links, err := c.GetDeviceByIP(ctx, ip)
	if err != nil {
		return DeviceDetails{}, err
	}
	return c.getDeviceDetails(ctx, dev, links, cfg.getDetails)
}
