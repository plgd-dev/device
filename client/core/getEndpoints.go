package core

import (
	"github.com/plgd-dev/device/schema"
)

func (d *Device) GetEndpoints() schema.Endpoints {
	d.lock.Lock()
	getEndpoints := d.getEndpoints
	d.lock.Unlock()
	if getEndpoints != nil {
		return getEndpoints()
	}
	return nil
}
