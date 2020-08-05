package local

import (
	"context"
)

// DisownDevice disowns a device.
// For unsecure device it calls factory reset.
// For secure device it disowns.
func (c *Client) DisownDevice(ctx context.Context, deviceID string) error {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	ok, err := d.IsSecured(ctx, links)
	if err != nil {
		return err
	}
	if !ok {
		return d.FactoryReset(ctx, links)
	}

	return d.Disown(ctx, links)
}
