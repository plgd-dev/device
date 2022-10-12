package client

import (
	"context"
)

// DisownDevice disowns a device.
// For unsecure device it calls factory reset.
// For secure device it disowns.
func (c *Client) DisownDevice(ctx context.Context, deviceID string, opts ...CommonCommandOption) error {
	cfg := applyCommonOptions(opts...)
	d, links, err := c.GetDeviceByMulticast(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}
	defer func() {
		dev, ok := c.deviceCache.LoadAndDeleteDevice(ctx, d.DeviceID())
		if ok {
			dev.Close(ctx)
		}
	}()

	ok := d.IsSecured()
	if !ok {
		return d.FactoryReset(ctx, links)
	}

	return d.Disown(ctx, links)
}
