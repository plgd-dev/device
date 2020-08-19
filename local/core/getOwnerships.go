package core

import (
	"context"

	"github.com/plgd-dev/go-coap/v2/udp/client"
	"github.com/plgd-dev/sdk/schema"
)

// OwnershipHandler conveys device ownership and errors during discovery.
type OwnershipHandler interface {
	// Handle gets a device ownership.
	Handle(ctx context.Context, doxm schema.Doxm)
	// Error gets errors during discovery.
	Error(err error)
}

// GetOwnerships discovers device's ownerships using a CoAP multicast request via UDP.
// It waits for device responses until the context is canceled.
func (c *Client) GetOwnerships(
	ctx context.Context,
	status DiscoverOwnershipStatus,
	handler OwnershipHandler,
) error {
	multicastConn := DialDiscoveryAddresses(ctx, c.discoveryConfiguration, c.errFunc)
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()
	h := newOwnershipHandler(handler)
	return DiscoverDeviceOwnership(ctx, multicastConn, status, h)
}

func newOwnershipHandler(
	h OwnershipHandler,
) *ownershipHandler {
	return &ownershipHandler{
		handler: h,
	}
}

type ownershipHandler struct {
	handler OwnershipHandler
}

func (h *ownershipHandler) Handle(ctx context.Context, conn *client.ClientConn, doxm schema.Doxm) {
	conn.Close()
	h.handler.Handle(ctx, doxm)
}

func (h *ownershipHandler) Error(err error) {
	h.handler.Error(err)
}
