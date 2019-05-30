package local

import (
	"context"

	coap "github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/sdk/local/device"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"

	gocoap "github.com/go-ocf/go-coap"
)

// GetDeviceOwnership discovers devices using a CoAP multicast request via UDP.
func (c *Client) GetDeviceOwnership(ctx context.Context, owned bool, handler DeviceHandler) error {
	return resource.DiscoverDeviceOwnership(ctx, c.conn, owned, c.newDiscoverDeviceOwnershipHandler(handler))
}

type discoverDeviceOwnershipHandler struct {
	handler DeviceHandler
	factory ResourceClientFactory
}

func (c *Client) newDiscoverDeviceOwnershipHandler(h DeviceHandler) *discoverDeviceOwnershipHandler {
	return &discoverDeviceOwnershipHandler{handler: h, factory: c.factory}
}

func (h *discoverDeviceOwnershipHandler) Handle(ctx context.Context, client *gocoap.ClientConn, ownership schema.Doxm) {
	links := schema.DeviceLinks{ID: ownership.DeviceId}
	c, err := h.factory.NewClient(client, links, coap.CBORCodec{})
	if err != nil {
		h.handler.Error(err)
		return
	}
	h.handler.Handle(ctx, device.NewClient(c, links, ownership))
}

func (h *discoverDeviceOwnershipHandler) Error(err error) {
	h.handler.Error(err)
}
