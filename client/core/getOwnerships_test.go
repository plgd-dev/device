package core_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/client/core/otm"
	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func testGetOwnerShips(ctx context.Context, t *testing.T, c *Client, ownStatus core.DiscoverOwnershipStatus, found bool) {
	timeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var h testOwnerShipHandler
	err := c.GetOwnerships(timeout, core.DefaultDiscoveryConfiguration(), ownStatus, &h)
	require.NoError(t, err)
	assert.Equal(t, found, h.anyFound.Load())
}

func ownDevice(ctx context.Context, t *testing.T, c *Client, deviceID string) func() {
	signer, err := NewTestSigner()
	require.NoError(t, err)
	device, err := c.GetDeviceByMulticast(ctx, deviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	defer func() {
		errClose := device.Close(ctx)
		require.NoError(t, errClose)
	}()
	eps := device.GetEndpoints()
	links, err := device.GetResourceLinks(ctx, eps)
	require.NoError(t, err)

	err = device.Own(ctx, links, []otm.Client{c.mfgOtm}, core.WithSetupCertificates(signer.Sign))
	require.NoError(t, err)

	links, err = device.GetResourceLinks(ctx, eps)
	require.NoError(t, err)
	return func() {
		err = device.Disown(ctx, links)
		require.NoError(t, err)
	}
}

func TestGetOwnerships(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errClose := c.Close()
		require.NoError(t, errClose)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	testGetOwnerShips(ctx, t, c, core.DiscoverAllDevices, true)
	testGetOwnerShips(ctx, t, c, core.DiscoverDisownedDevices, true)
	testGetOwnerShips(ctx, t, c, core.DiscoverOwnedDevices, false)

	disown := ownDevice(ctx, t, c, secureDeviceID)
	defer disown()

	testGetOwnerShips(ctx, t, c, core.DiscoverDisownedDevices, false)
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
