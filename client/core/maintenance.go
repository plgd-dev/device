package core

import (
	"context"
	"fmt"
	"net/http"

	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/maintenance"
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
	err := d.updateMaintenanceResource(ctx, links, maintenance.MaintenanceUpdateRequest{
		FactoryReset: true,
	})
	if connectionWasClosed(ctx, err) {
		// connection was closed by disown so we don't report error just log it.
		d.cfg.ErrFunc(err)
		return nil
	}
	return err
}

func (d *Device) updateMaintenanceResource(
	ctx context.Context,
	links schema.ResourceLinks,
	req maintenance.MaintenanceUpdateRequest,
) (ret error) {
	links = links.GetResourceLinks(maintenance.ResourceType)
	if len(links) == 0 {
		return MakeUnavailable(fmt.Errorf("cannot find '%v' in %+v", maintenance.ResourceType, links))
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
