package local

import (
	"context"

	"github.com/go-ocf/sdk/schema"

	gocoap "github.com/go-ocf/go-coap"
)

// DeviceHandler conveys device connections and errors during discovery.
type DeviceHandler interface {
	// Handle gets a device connection and is responsible for closing it.
	Handle(ctx context.Context, device *Device)
	// Error gets errors during discovery.
	Error(err error)
}

// GetDevices discovers devices using a CoAP multicast request via UDP.
// Device resources can be queried in DeviceHandler using device.Client,
func (c *Client) GetDevices(ctx context.Context, handler DeviceHandler) error {
	return DiscoverDevices(ctx, c.conn, newDiscoveryHandler(c.tlsConfig, c.conn, handler))
}

func newDiscoveryHandler(tlsConfig *TLSConfig, multicastConn []*gocoap.MulticastClientConn, h DeviceHandler) *discoveryHandler {

	return &discoveryHandler{tlsConfig: tlsConfig, multicastConn: multicastConn, handler: h}
}

type discoveryHandler struct {
	multicastConn []*gocoap.MulticastClientConn
	tlsConfig     *TLSConfig
	handler       DeviceHandler
}

func (h *discoveryHandler) Handle(ctx context.Context, conn *gocoap.ClientConn, links schema.DeviceLinks) {
	h.handler.Handle(ctx, NewDevice(links, conn, h.multicastConn, h.tlsConfig))
}

func (h *discoveryHandler) Error(err error) {
	h.handler.Error(err)
}
