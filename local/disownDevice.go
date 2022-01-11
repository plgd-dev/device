package local

import (
	"context"

	"github.com/plgd-dev/sdk/local/core"
)

// DisownDevice disowns a device.
// For unsecure device it calls factory reset.
// For secure device it disowns.
func (c *Client) DisownDevice(ctx context.Context, deviceID string, opts ...CommonCommandOption) error {
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
	c.deviceCache.RemoveDevice(ctx, d.DeviceID(), d)
	defer d.Release(ctx)

	ok := d.IsSecured()
	if !ok {
		return d.FactoryReset(ctx, links)
	}

	return d.Disown(ctx, links)
}
