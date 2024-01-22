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
	"fmt"
	"net/http"

	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/maintenance"
)

func (d *Device) Reboot(
	ctx context.Context,
	links schema.ResourceLinks,
	options ...coap.OptionFunc,
) error {
	return d.updateMaintenanceResource(ctx, links, maintenance.MaintenanceUpdateRequest{
		Reboot: true,
	}, options...)
}

func (d *Device) FactoryReset(
	ctx context.Context,
	links schema.ResourceLinks,
	options ...coap.OptionFunc,
) error {
	err := d.updateMaintenanceResource(ctx, links, maintenance.MaintenanceUpdateRequest{
		FactoryReset: true,
	}, options...)
	if connectionWasClosed(ctx, err) {
		// connection was closed by disown so we don't report error just log it.
		d.cfg.Logger.Debug(err.Error())
		return nil
	}
	return err
}

func (d *Device) updateMaintenanceResource(
	ctx context.Context,
	links schema.ResourceLinks,
	req maintenance.MaintenanceUpdateRequest,
	options ...coap.OptionFunc,
) (ret error) {
	links = links.GetResourceLinks(maintenance.ResourceType)
	if len(links) == 0 {
		return MakeUnavailable(fmt.Errorf("cannot find '%v' in %+v", maintenance.ResourceType, links))
	}
	var resp maintenance.Maintenance
	err := d.UpdateResource(ctx, links[0], req, &resp, options...)
	if err != nil {
		return err
	}
	if resp.LastHTTPError >= http.StatusBadRequest {
		str := http.StatusText(resp.LastHTTPError)
		defer func() {
			if r := recover(); r != nil {
				ret = fmt.Errorf("returns HTTP code %v", resp.LastHTTPError)
			}
		}()
		return fmt.Errorf(str)
	}
	return nil
}
