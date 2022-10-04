package client

import (
	"context"
)

func (c *Client) DeleteDevice(ctx context.Context, deviceID string) (bool, error) {
	dev, ok := c.deviceCache.LoadAndDeleteDevice(ctx, deviceID)
	if !ok {
		return false, nil
	}
	err := dev.Close(ctx)
	if err != nil {
		c.errors(err)
	}
	return true, nil
}
