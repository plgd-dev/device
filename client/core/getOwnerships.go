package core

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/schema/doxm"
	"github.com/plgd-dev/go-coap/v2/udp/client"
)

// OwnershipHandler conveys device ownership and errors during discovery.
type OwnershipHandler interface {
	// Handle gets a device ownership.
	Handle(ctx context.Context, doxm doxm.Doxm)
	// Error gets errors during discovery.
	Error(err error)
}

// GetOwnerships discovers device's ownerships using a CoAP multicast request via UDP.
// It waits for device responses until the context is canceled.
func (c *Client) GetOwnerships(
	ctx context.Context,
	discoveryConfiguration DiscoveryConfiguration,
	status DiscoverOwnershipStatus,
	handler OwnershipHandler,
) error {
	multicastConn, err := DialDiscoveryAddresses(ctx, discoveryConfiguration, c.errFunc)
	if err != nil {
		return MakeInvalidArgument(fmt.Errorf("could not get the ownerships: %w", err))
	}
	defer func() {
		for _, conn := range multicastConn {
			if errC := conn.Close(); errC != nil {
				c.errFunc(fmt.Errorf("get ownership error: cannot close connection(%s): %w", conn.mcastaddr, errC))
			}
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

func (h *ownershipHandler) Handle(ctx context.Context, conn *client.ClientConn, doxm doxm.Doxm) {
	if errC := conn.Close(); errC != nil {
		h.Error(fmt.Errorf("ownership handler cannot close connection: %w", errC))
	}
	h.handler.Handle(ctx, doxm)
}

func (h *ownershipHandler) Error(err error) {
	h.handler.Error(err)
}
