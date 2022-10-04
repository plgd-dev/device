package client

import (
	"context"
)

func (c *Client) OffboardDevice(ctx context.Context, deviceID string, opts ...CommonCommandOption) error {
	cfg := applyCommonOptions(opts...)
	d, links, err := c.GetDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}

	return setCloudResource(ctx, links, d, "", "", "", "")
}
