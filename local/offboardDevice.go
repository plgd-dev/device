package local

import (
	"context"
	"fmt"
)

func (c *Client) OffboardDevice(
	ctx context.Context,
	deviceID string,
) error {
	const errMsg = "cannot offboard device %v: %v"
	return c.onboardOffboardDevice(ctx, deviceID, "", "", "", func(err error) error {
		return fmt.Errorf(errMsg, deviceID, err)
	})
}
