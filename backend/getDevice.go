package backend

import (
	"context"
	"fmt"
)

// GetDevice retrieves device details from the backend.
func (c *Client) GetDevice(
	ctx context.Context,
	deviceID string,
) (DeviceDetails, error) {
	devices, err := c.GetDevices(ctx, WithDeviceIDs(deviceID))
	if err != nil {
		return DeviceDetails{}, err
	}
	if len(devices) == 0 {
		return DeviceDetails{}, fmt.Errorf("not found")
	}
	return devices[deviceID], nil
}
