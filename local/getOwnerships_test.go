package local_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ocf "github.com/go-ocf/sdk/local"
	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/test"

	"github.com/stretchr/testify/require"
)

func testGetOwnerShips(ctx context.Context, t *testing.T, c *Client, ownStatus ocf.DiscoverOwnershipStatus, found bool) {
	timeout, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var h testOwnerShipHandler
	err := c.GetOwnerships(timeout, ownStatus, &h)
	require.NoError(t, err)
	assert.Equal(t, found, h.anyFound)
}

func TestGetOwnerships(t *testing.T) {
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	testGetOwnerShips(ctx, t, c, ocf.DiscoverAllDevices, true)
	testGetOwnerShips(ctx, t, c, ocf.DiscoverDisownedDevices, true)
	testGetOwnerShips(ctx, t, c, ocf.DiscoverOwnedDevices, false)

	deviceID := test.TestSecureDeviceID
	device, links, err := c.GetDevice(ctx, deviceID)
	require.NoError(t, err)
	defer device.Close(ctx)

	err = device.Own(ctx, links, c.otm)
	require.NoError(t, err)

	testGetOwnerShips(ctx, t, c, ocf.DiscoverDisownedDevices, false)
	testGetOwnerShips(ctx, t, c, ocf.DiscoverOwnedDevices, true)

	err = device.Disown(ctx, links)
	require.NoError(t, err)
}

type testOwnerShipHandler struct {
	anyFound bool
}

func (h *testOwnerShipHandler) Handle(ctx context.Context, doxm schema.Doxm) {
	fmt.Printf("testOwnerShipHandler.Handle: %+v\n", doxm)
	h.anyFound = true
}

func (h *testOwnerShipHandler) Error(err error) {
}
