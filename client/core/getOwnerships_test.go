package core_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	ocf "github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/schema/doxm"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func testGetOwnerShips(ctx context.Context, t *testing.T, c *Client, ownStatus ocf.DiscoverOwnershipStatus, found bool) {
	timeout, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	var h testOwnerShipHandler
	err := c.GetOwnerships(timeout, ocf.DefaultDiscoveryConfiguration(), ownStatus, &h)
	require.NoError(t, err)
	assert.Equal(t, found, h.anyFound.Load())
}

func TestGetOwnerships(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.TestSecureDeviceName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	testGetOwnerShips(ctx, t, c, ocf.DiscoverAllDevices, true)
	testGetOwnerShips(ctx, t, c, ocf.DiscoverDisownedDevices, true)
	testGetOwnerShips(ctx, t, c, ocf.DiscoverOwnedDevices, false)

	deviceID := secureDeviceID
	device, err := c.GetDeviceByMulticast(ctx, deviceID, ocf.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	defer device.Close(ctx)
	eps := device.GetEndpoints()
	links, err := device.GetResourceLinks(ctx, eps)
	require.NoError(t, err)

	err = device.Own(ctx, links, c.mfgOtm)
	require.NoError(t, err)

	links, err = device.GetResourceLinks(ctx, eps)
	require.NoError(t, err)

	testGetOwnerShips(ctx, t, c, ocf.DiscoverDisownedDevices, false)

	err = device.Disown(ctx, links)
	require.NoError(t, err)
}

type testOwnerShipHandler struct {
	anyFound atomic.Bool
}

func (h *testOwnerShipHandler) Handle(ctx context.Context, doxm doxm.Doxm) {
	h.anyFound.Store(true)
}

func (h *testOwnerShipHandler) Error(err error) {
	fmt.Print(err)
}
