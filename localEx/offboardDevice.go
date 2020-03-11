package localEx

import (
	"context"
)

func (c *Client) OffboardDevice(ctx context.Context, deviceID string) error {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	// TODO remove workaround
	return setCloudResource(ctx, links, d, "", "", "", "")
	/*
		ok, err := d.IsSecured(ctx, links)
		if err != nil {
			return err
		}
		if ok {
			return d.Offboard(ctx)
		}
		return d.OffboardInsecured(ctx)
	*/
}
