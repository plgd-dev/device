package client

import (
	"context"
)

func (c *Client) FactoryReset(ctx context.Context, deviceID string, opts ...CommonCommandOption) error {
	cfg := applyCommonOptions(opts...)
	d, links, err := c.GetRefDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	return d.FactoryReset(ctx, links)
}

func (c *Client) Reboot(ctx context.Context, deviceID string, opts ...CommonCommandOption) error {
	cfg := applyCommonOptions(opts...)
	d, links, err := c.GetRefDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	return d.Reboot(ctx, links)
}