// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/pstat"
)

func connectionWasClosed(ctx context.Context, err error) bool {
	if ctx.Err() == nil && errors.Is(err, context.Canceled) {
		return true
	}
	return false
}

// DisownDevice removes ownership of device
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
			d.cfg.Logger.Debug(fmt.Errorf("device disown error: %w", err).Error())
			return nil
		}

		return MakeInternal(cannotDisownErr(err))
	}
	if errC := d.Close(ctx); errC != nil {
		d.cfg.Logger.Debug(fmt.Errorf("device disown error: %w", errC).Error())
	}
	return nil
}
