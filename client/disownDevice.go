package client

import (
	"context"

	"github.com/plgd-dev/device/client/core"
)

func (c *Client) removeTemporaryDeviceFromCache(ctx context.Context, d *core.Device) {
	if d.FoundByIP() != "" {
		// device is found by IP, so it is not temporary
		return
	}
	deleteDeviceNotFoundByIP(ctx, c.deviceCache, d)
}

// DisownDevice disowns a device.
// For unsecure device it calls factory reset.
// For secure device it disowns.
func (c *Client) DisownDevice(ctx context.Context, deviceID string, opts ...CommonCommandOption) error {
	cfg := applyCommonOptions(opts...)
	d, links, err := c.GetDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}
	defer c.removeTemporaryDeviceFromCache(ctx, d)

	ok := d.IsSecured()
	if !ok {
		return d.FactoryReset(ctx, links)
	}

	return d.Disown(ctx, links)
}
