package core

import (
	"context"
	"fmt"

	"github.com/go-ocf/sdk/schema"
)

// DisownDevice remove ownership of device
func (d *Device) Disown(
	ctx context.Context,
	links schema.ResourceLinks,
) error {
	const errMsg = "cannot disown: %w"

	ownership, err := d.GetOwnership(ctx)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	sdkID, err := d.GetSdkOwnerID()
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	if ownership.OwnerID != sdkID {
		if ownership.OwnerID == "00000000-0000-0000-0000-000000000000" {
			return nil
		}
		return fmt.Errorf(errMsg, fmt.Errorf("device is owned by %v, not by %v", ownership.OwnerID, sdkID))
	}

	setResetProvisionState := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RESET,
		},
	}

	link, err := GetResourceLink(links, "/oic/sec/pstat")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	err = d.UpdateResource(ctx, link, setResetProvisionState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	return nil
}
