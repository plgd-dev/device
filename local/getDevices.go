package local

import (
	"context"

	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"

	gocoap "github.com/go-ocf/go-coap"
)

// DeviceHandler can use the client to query details about device's resources.
type DeviceHandler interface {
	Handle(ctx context.Context, device *Device)
	Error(err error)
}

// GetDevices discovers devices using a CoAP multicast request via UDP.
// Device resources can be queried in DeviceHandler using device.Client,
// which also caches resource links and pools connections.
// An empty typeFilter queries all resource types.
// Note: len(typeFilter) > 1 does not work with Iotivity 1.3 which responds with BadRequest.
func (c *Client) GetDevices(ctx context.Context, typeFilter []string, handler DeviceHandler) error {
	options := make([]coap.OptionFunc, 0, len(typeFilter))
	for _, t := range typeFilter {
		options = append(options, coap.WithResourceType(t))
	}
	return resource.DiscoverDevices(ctx, c.conn, newDiscoveryHandler(handler), options...)
}

func newDiscoveryHandler(h DeviceHandler) *discoveryHandler {
	return &discoveryHandler{handler: h}
}

type discoveryHandler struct {
	handler DeviceHandler
}

func (h *discoveryHandler) Handle(ctx context.Context, conn *gocoap.ClientConn, links schema.DeviceLinks) {
	h.handler.Handle(ctx, NewDevice(links, conn))
}

func (h *discoveryHandler) Error(err error) {
	h.handler.Error(err)
}
