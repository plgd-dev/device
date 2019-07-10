package local

import (
	"context"
	"fmt"

	"github.com/go-ocf/sdk/schema"
)

// DisownDevice remove ownership of device
func (d *Device) Disown(
	ctx context.Context,
) error {
	const errMsg = "cannot disown: %v"

	ownership, err := d.GetOwnership(ctx)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	sdkID, err := d.GetSdkDeviceID()
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	if ownership.DeviceOwner != sdkID {
		return fmt.Errorf(errMsg, fmt.Sprintf("device is owned by %v, not by %v", ownership.DeviceOwner, sdkID))
	}

	setResetProvisionState := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RESET,
		},
	}

	err = d.UpdateResource(ctx, "/oic/sec/pstat", setResetProvisionState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	return nil
}
