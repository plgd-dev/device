package local

import (
	"context"
)

// OffboardDevice for unsecure device it reset attributes, for secure device it calls Disown.
func (c *Client) OffboardDevice(ctx context.Context, deviceID string) error {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	ok, err := d.IsSecured(ctx, links)
	if err != nil {
		return err
	}
	if ok {
		return d.Disown(ctx, links)
	}
	return setCloudResource(ctx, links, d, "", "", "", "")
}
