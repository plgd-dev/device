package local

import (
	"context"

	"github.com/plgd-dev/sdk/local/core"
)

func (c *Client) FactoryReset(ctx context.Context, deviceID string, opts ...CommonCommandOption) error {
	cfg := commonCommandOptions{
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnCommonCommand(cfg)
	}
	d, links, err := c.GetRefDevice(ctx, deviceID, WithDiscoveryConfigration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	return d.FactoryReset(ctx, links)
}

func (c *Client) Reboot(ctx context.Context, deviceID string, opts ...CommonCommandOption) error {
	cfg := commonCommandOptions{
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnCommonCommand(cfg)
	}
	d, links, err := c.GetRefDevice(ctx, deviceID, WithDiscoveryConfigration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	return d.Reboot(ctx, links)
}
