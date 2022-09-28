package client

import (
	"context"
)

func (c *Client) RemoveDevice(ctx context.Context, deviceID string) (bool, error) {
	d, _, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return false, err
	}
	defer d.Release(ctx)

	return c.deviceCache.RemoveDevice(ctx, deviceID, d), nil
}
