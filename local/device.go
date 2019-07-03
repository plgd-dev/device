package local

import (
	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/sdk/schema"
)

type Device struct {
	schema.DeviceLinks
	conn *gocoap.ClientConn
}

func NewDevice(links schema.DeviceLinks, conn *gocoap.ClientConn) *Device {
	return &Device{
		DeviceLinks: links,

		conn: conn,
	}
}

// Close closes open connections to the device.
func (d *Device) Close() {
	if d.conn != nil {
		d.conn.Close()
	}
}

// GetResourceLinks returns all resource links.
func (d *Device) GetResourceLinks() []schema.ResourceLink {
	return d.Links
}

// GetDeviceLinks returns device links.
func (d *Device) GetDeviceLinks() schema.DeviceLinks {
	return d.DeviceLinks
}
