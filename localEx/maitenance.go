package localEx

import "context"

func (c *Client) FactoryReset(ctx context.Context, deviceID string) error {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	return d.FactoryReset(ctx, links)
}

func (c *Client) Reboot(ctx context.Context, deviceID string) error {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	return d.Reboot(ctx, links)
}
