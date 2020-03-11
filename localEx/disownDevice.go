package localEx

import (
	"context"
)

// DisownDevice disowns a device.
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
		// don't disown insecure device
		return nil
	}

	return d.Disown(ctx, links)
}
