package local

import (
	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/net/coap"
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

// Connection returns a connection
func (d *Device) connection(endpoint string) *coap.Client {
	return coap.NewClient(d.conn)
}

func (d *Device) DeviceID() string                        { return d.ID }
func (d *Device) GetResourceLinks() []schema.ResourceLink { return d.Links }
func (d *Device) GetDeviceLinks() schema.DeviceLinks      { return d.DeviceLinks }

// GetEndpoints returns endpoints for a resource type.
// The endpoints are returned in order of priority.
func (d *Device) GetEndpoints(resourceType string) []schema.Endpoint {
	return d.GetEndpoints(resourceType)
}
