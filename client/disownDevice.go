package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/kit/v2/log"
)

// DisownDevice disowns a device.
// For unsecure device it calls factory reset.
// For secure device it disowns.
func (c *Client) DisownDevice(ctx context.Context, deviceID string, opts ...CommonCommandOption) error {
	cfg := applyCommonOptions(opts...)
	d, links, err := c.GetRefDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}
	log.Info("DisownDevice 1")
	c.deviceCache.RemoveDevice(ctx, d.DeviceID(), d)
	defer func() {
		if errRelease := d.Release(ctx); errRelease != nil {
			c.errors(fmt.Errorf("disown device error: %w", errRelease))
		}
	}()

	log.Info("DisownDevice 2")
	ok := d.IsSecured()
	if !ok {
		return d.FactoryReset(ctx, links)
	}
	log.Info("DisownDevice 3")

	return d.Disown(ctx, links)
}
