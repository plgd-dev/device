package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/pstat"
	"github.com/plgd-dev/kit/v2/log"
)

func connectionWasClosed(ctx context.Context, err error) bool {
	if ctx.Err() == nil && errors.Is(err, context.Canceled) {
		return true
	}
	return false
}

// DisownDevice remove ownership of device
func (d *Device) Disown(
	ctx context.Context,
	links schema.ResourceLinks,
) error {
	const errMsg = "cannot disown: %w"

	log.Info("Disown 1")

	ownership, err := d.GetOwnership(ctx, links)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	log.Info("Disown 2")

	sdkID, err := d.GetSdkOwnerID()
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	log.Info("Disown 3")

	if ownership.OwnerID != sdkID {
		if ownership.OwnerID == uuid.Nil.String() {
			return nil
		}
		return MakePermissionDenied(fmt.Errorf(errMsg, fmt.Errorf("device is owned by %v, not by %v", ownership.OwnerID, sdkID)))
	}

	setResetProvisionState := pstat.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &pstat.DeviceOnboardingState{
			CurrentOrPendingOperationalState: pstat.OperationalState_RESET,
		},
	}

	log.Info("Disown 4")

	link, err := GetResourceLink(links, pstat.ResourceURI)
	if err != nil {
		return MakeInternal(fmt.Errorf(errMsg, err))
	}
	link.Endpoints = link.Endpoints.FilterSecureEndpoints()

	log.Info("Disown 5")

	err = d.UpdateResource(ctx, link, setResetProvisionState, nil)
	if err != nil {
		if connectionWasClosed(ctx, err) {
			// connection was closed by disown so we don't report error just log it.
			d.cfg.errFunc(fmt.Errorf(errMsg, err))
			return nil
		}

		return MakeInternal(fmt.Errorf(errMsg, err))
	}

	log.Info("Disown 6")
	d.Close(ctx)

	log.Info("Disown 7")

	return nil
}
