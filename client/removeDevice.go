package client

import (
	"context"
	"fmt"
)

func (c *Client) RemoveDevice(ctx context.Context, deviceID string) (bool, error) {
	var removed bool
	// TODO: temporary workaroud to remove devices from both caches
	// will be fixed by combining temporary/permanent cache into a single one
	for {
		refDev, found := c.deviceCache.GetDevice(ctx, deviceID)
		if !found {
			if removed {
				break
			}
			return false, fmt.Errorf("Device not found")
		}
		removed = true
		defer refDev.Release(ctx)
		if !c.deviceCache.RemoveDevice(ctx, deviceID, refDev) {
			break
		}
	}

	return removed, nil
}
