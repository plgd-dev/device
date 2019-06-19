package local

import (
	"context"
	"fmt"
)

func (c *Client) OffboardInsecuredDevice(
	ctx context.Context,
	deviceID string,
) error {
	const errMsg = "cannot offboard device %v: %v"
	ok, err := c.isSecuredDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	if ok {
		return fmt.Errorf(errMsg, deviceID, "is secured device")
	}

	err = c.onboardOffboardInsecuredDevice(ctx, deviceID, "", "", "")
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}
	return nil
}

func (c *Client) OffboardDevice(
	ctx context.Context,
	deviceID string,
) error {
	const errMsg = "cannot offboard device %v: %v"
	ok, err := c.isSecuredDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	if !ok {
		return fmt.Errorf(errMsg, deviceID, "is insecured device")
	}

	return c.DisownDevice(ctx, deviceID)
}
