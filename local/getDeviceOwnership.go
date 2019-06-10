package local

import (
	"context"

	"github.com/go-ocf/sdk/local/resource"
)

// GetDeviceOwnership discovers devices using a CoAP multicast request via UDP.
func (c *Client) GetDeviceOwnership(ctx context.Context, status resource.DiscoverOwnershipStatus, handler resource.DiscoverDeviceOwnershipHandler) error {
	return resource.DiscoverDeviceOwnership(ctx, c.conn, status, handler)
}
