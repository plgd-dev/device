package core

import (
	"context"
	"fmt"
	"net/http"

	"github.com/plgd-dev/sdk/schema"
	"github.com/plgd-dev/sdk/schema/maintenance"
)

func (d *Device) Reboot(
	ctx context.Context,
	links schema.ResourceLinks,
) error {
	return d.updateMaintenanceResource(ctx, links, maintenance.MaintenanceUpdateRequest{
		Reboot: true,
	})
}

func (d *Device) FactoryReset(
	ctx context.Context,
	links schema.ResourceLinks,
) error {
	return d.updateMaintenanceResource(ctx, links, maintenance.MaintenanceUpdateRequest{
		FactoryReset: true,
	})
}

func (d *Device) updateMaintenanceResource(
	ctx context.Context,
	links schema.ResourceLinks,
	req maintenance.MaintenanceUpdateRequest,
) (ret error) {
	links = links.GetResourceLinks(maintenance.MaintenanceResourceType)
	if len(links) == 0 {
		return fmt.Errorf("cannot find '%v' in %+v", maintenance.MaintenanceResourceType, links)
	}
	var resp maintenance.Maintenance
	err := d.UpdateResource(ctx, links[0], req, &resp)
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
