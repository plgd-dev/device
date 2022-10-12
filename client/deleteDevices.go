package client

import (
	"context"
	"fmt"
)

func (c *Client) DeleteDevice(ctx context.Context, deviceID string) bool {
	devs := c.DeleteDevices(ctx, []string{deviceID})
	return len(devs) > 0
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
			c.errors(fmt.Errorf("can't close device %v during deleting device from the cache: %w", d.DeviceID(), err))
		}
	}
	return deviceIDs
}
