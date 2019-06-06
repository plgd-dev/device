package local

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"
)

// DisownDevice remove ownership of device
func (c *Client) DisownDevice(
	ctx context.Context,
	deviceID string,
	discoveryTimeout time.Duration,
) error {
	const errMsg = "cannot disown device %v: %v"

	client, err := c.ownDeviceFindClient(ctx, deviceID, discoveryTimeout, resource.DiscoverAllDevices)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	ownership := client.GetOwnership()
	if !ownership.Owned {
		return fmt.Errorf(errMsg, deviceID, "device is not owned")
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

	return nil
}
