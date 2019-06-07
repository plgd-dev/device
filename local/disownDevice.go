package local

import (
	"context"
	"fmt"

	"github.com/go-ocf/sdk/local/device"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"
)

// DisownDevice remove ownership of device
func (c *Client) DisownDevice(
	ctx context.Context,
	deviceID string,
) error {
	const errMsg = "cannot disown device %v: %v"

	client, err := c.ownDeviceFindClient(ctx, deviceID, resource.DiscoverAllDevices)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	ownership := client.GetOwnership()
	if !ownership.Owned {
		return fmt.Errorf(errMsg, deviceID, "device is not owned")
	}

	var deviceClient *device.Client
	err = c.GetDevice(ctx, deviceID, nil, &deviceClient)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	sdkID, err := c.GetSdkID()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	if ownership.DeviceOwner != sdkID {
		return fmt.Errorf(errMsg, deviceID, fmt.Sprintf("device is owned by %v, not by %v", ownership.DeviceOwner, sdkID))
	}

	setResetProvisionState := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RESET,
		},
	}

	err = c.UpdateResourceCBOR(ctx, deviceID, "/oic/sec/pstat", setResetProvisionState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	c.CloseConnections(deviceClient.GetDeviceLinks())

	return nil
}
