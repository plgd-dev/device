package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/pstat"
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
	cannotDisownErr := func(err error) error {
		return fmt.Errorf("cannot disown: %w", err)
	}

	ownership, err := d.GetOwnership(ctx, links)
	if err != nil {
		return cannotDisownErr(err)
	}

	sdkID, err := d.GetSdkOwnerID()
	if err != nil {
		return cannotDisownErr(err)
	}

	if ownership.OwnerID != sdkID {
		if ownership.OwnerID == uuid.Nil.String() {
			return nil
		}
		return MakePermissionDenied(cannotDisownErr(fmt.Errorf("device is owned by %v, not by %v", ownership.OwnerID, sdkID)))
	}

	setResetProvisionState := pstat.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &pstat.DeviceOnboardingState{
			CurrentOrPendingOperationalState: pstat.OperationalState_RESET,
		},
	}

	link, err := GetResourceLink(links, pstat.ResourceURI)
	if err != nil {
		return MakeInternal(cannotDisownErr(err))
	}
	link.Endpoints = link.Endpoints.FilterSecureEndpoints()

	err = d.UpdateResource(ctx, link, setResetProvisionState, nil)
	if err != nil {
		if connectionWasClosed(ctx, err) {
			// connection was closed by disown so we don't report error just log it.
			d.cfg.ErrFunc(cannotDisownErr(err))
			return nil
		}

		return MakeInternal(cannotDisownErr(err))
	}
	d.Close(ctx)

	return nil
}
