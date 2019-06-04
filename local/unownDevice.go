package local

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"
)

// UnownDevice remove ownership of device
func (c *Client) UnownDevice(
	ctx context.Context,
	deviceID string,
	discoveryTimeout time.Duration,
) error {
	const errMsg = "cannot unown device %v: %v"

	client, err := c.ownDeviceFindClient(ctx, deviceID, discoveryTimeout, resource.DiscoverAllDevices)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	ownership := client.GetOwnership()
	if !ownership.Owned {
		return fmt.Errorf(errMsg, deviceID, "device is not owned")
	}

	sdkId, err := c.GetSdkId()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	if ownership.DeviceOwner != sdkId {
		return fmt.Errorf(errMsg, deviceID, fmt.Sprintf("device is owned by %v, not by %v", ownership.DeviceOwner, sdkId))
	}

	setResetProvisionState := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RESET,
		},
	}

	err = c.UpdateResourceCBOR(ctx, deviceID, "/oic/sec/pstat", "", setResetProvisionState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}
	return nil
}
