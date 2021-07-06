package core

import (
	"github.com/plgd-dev/sdk/schema"
)

func (d *Device) GetEndpoints() schema.Endpoints {
	return d.endpoints
}
