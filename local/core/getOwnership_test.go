package core_test

import (
	"context"
	"testing"
	"time"

	ocf "github.com/plgd-dev/sdk/local/core"
	"github.com/plgd-dev/sdk/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOwnership(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.TestSecureDeviceName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	deviceID := secureDeviceID
	device, err := c.GetDeviceByMulticast(ctx, deviceID, ocf.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	defer device.Close(ctx)
	eps := device.GetEndpoints()
	links, err := device.GetResourceLinks(ctx, eps)
	require.NoError(t, err)

	// before own
	got, err := device.GetOwnership(ctx, links)
	require.NoError(t, err)
	assert.False(t, got.Owned)

	err = device.Own(ctx, links, c.mfgOtm)
	require.NoError(t, err)

	// after own
	got, err = device.GetOwnership(ctx, links)
	require.NoError(t, err)
	assert.True(t, got.Owned)
	assert.Equal(t, CertIdentity, got.OwnerID)

	err = device.Disown(ctx, links)
	require.NoError(t, err)
}
