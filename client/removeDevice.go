package client

import (
	"context"
	"fmt"
)

func (c *Client) RemoveDevice(ctx context.Context, deviceID string) (bool, error) {
	refDev, found := c.deviceCache.GetDevice(deviceID)
	if !found {
		return false, fmt.Errorf("Device not found")
	}

	defer refDev.Release(ctx)
	removed := c.deviceCache.RemoveDevice(deviceID, refDev)

	return removed, nil
}
