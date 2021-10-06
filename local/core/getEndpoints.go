package core

import (
	"github.com/plgd-dev/sdk/v2/schema"
)

func (d *Device) GetEndpoints() schema.Endpoints {
	return d.endpoints
}
