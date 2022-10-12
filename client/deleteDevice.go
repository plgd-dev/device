package client

import (
	"context"
)

// DeleteResource deletes a device from the cache.
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) (bool, error) {
	dev, ok := c.deviceCache.LoadAndDeleteDevice(ctx, deviceID)
	if !ok {
		return false, nil
	}
	err := dev.Close(ctx)
	if err != nil {
		c.logger.Debugf("can't close device %v during deleting device from the cache: %v", deviceID, err)
	}
	return true, nil
}
