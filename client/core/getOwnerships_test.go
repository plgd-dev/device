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

func ownDevice(ctx context.Context, t *testing.T, c *Client, deviceID string) func() {
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
	return func() {
		err = device.Disown(ctx, links)
		require.NoError(t, err)
	}
}

func TestGetOwnerships(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.DevsimNetBridge)
	secureDeviceID1 := test.MustFindDeviceByName(test.DevsimNetHost)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	testGetOwnerShips(ctx, t, c, ocf.DiscoverAllDevices, true)
	testGetOwnerShips(ctx, t, c, ocf.DiscoverDisownedDevices, true)
	testGetOwnerShips(ctx, t, c, ocf.DiscoverOwnedDevices, false)

	disown0 := ownDevice(ctx, t, c, secureDeviceID)
	defer disown0()

	disown1 := ownDevice(ctx, t, c, secureDeviceID1)
	defer disown1()

	testGetOwnerShips(ctx, t, c, ocf.DiscoverDisownedDevices, false)

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
