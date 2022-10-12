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
)

func (c *Client) DeleteDevice(ctx context.Context, deviceID string) bool {
	devs := c.DeleteDevices(ctx, []string{deviceID})
	if len(devs) == 0 {
		return false
	}
	return true
}

// DeleteDevices deletes a device from the cache. If deviceIDFilter is empty, all devices are deleted.
func (c *Client) DeleteDevices(ctx context.Context, deviceIDFilter []string) []string {
	devs := c.deviceCache.LoadAndDeleteDevices(deviceIDFilter)
	if len(devs) == 0 {
		return nil
	}
	deviceIDs := make([]string, 0, len(devs))
	for _, d := range devs {
		deviceIDs = append(deviceIDs, d.DeviceID())
		err := d.Close(ctx)
		if err != nil {
			c.logger.Debugf("can't close device %v during deleting device from the cache: %v", d.DeviceID(), err)
		}
	}
	return deviceIDs
}
