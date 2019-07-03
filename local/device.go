package local

import "github.com/go-ocf/sdk/schema"

type Device struct {
	schema.DeviceLinks
}

// GetResourceLinks returns all resource links.
func (d *Device) GetResourceLinks() []schema.ResourceLink {
	return d.Links
}

// GetDeviceLinks returns device links.
func (d *Device) GetDeviceLinks() schema.DeviceLinks {
	return d.DeviceLinks
}
