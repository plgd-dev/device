package core

import (
	"github.com/plgd-dev/device/schema"
)

func (d *Device) GetEndpoints() schema.Endpoints {
	return d.getEndpoints()
}
