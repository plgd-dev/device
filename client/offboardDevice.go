package client

import (
	"context"
)

func (c *Client) OffboardDevice(ctx context.Context, deviceID string) error {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	return setCloudResource(ctx, links, d, "", "", "", "")
}
