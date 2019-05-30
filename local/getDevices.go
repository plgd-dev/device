package local

import (
	"context"

	coap "github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/sdk/local/device"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"

	gocoap "github.com/go-ocf/go-coap"
)

// DeviceHandler can use the client to query details about device's resources.
type DeviceHandler interface {
	Handle(ctx context.Context, client *device.Client)
	Error(err error)
}

// GetDevices discovers devices using a CoAP multicast request via UDP.
// Device resources can be queried in DeviceHandler using device.Client,
// which also caches resource links and pools connections.
// An empty typeFilter queries all resource types.
// Note: len(typeFilter) > 1 does not work with Iotivity 1.3 which responds with BadRequest.
func (c *Client) GetDevices(ctx context.Context, typeFilter []string, handler DeviceHandler) error {
	return resource.DiscoverDevices(ctx, c.conn, typeFilter, c.newDiscoveryHandler(handler))
}

func (c *Client) newDiscoveryHandler(h DeviceHandler) *discoveryHandler {
	return &discoveryHandler{handler: h, factory: c.factory}
}

type discoveryHandler struct {
	handler DeviceHandler
	factory ResourceClientFactory
}

func (h *discoveryHandler) Handle(ctx context.Context, client *gocoap.ClientConn, links schema.DeviceLinks) {
	c, err := h.factory.NewClient(client, links, coap.CBORCodec{})
	if err != nil {
		h.handler.Error(err)
		return
	}
	h.handler.Handle(ctx, device.NewClient(c, links, schema.Doxm{}))
}

func (h *discoveryHandler) Error(err error) {
	h.handler.Error(err)
}
