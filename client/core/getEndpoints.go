package core

import (
	"github.com/plgd-dev/device/v2/schema"
)

func (d *Device) GetEndpoints() schema.Endpoints {
	return d.getEndpoints()
}
