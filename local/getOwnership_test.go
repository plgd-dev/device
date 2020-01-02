package local_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOwnership(t *testing.T) {
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deviceID := testGetDeviceID(t, c.Client, true)
	device, links, err := c.GetDevice(ctx, deviceID)
	require.NoError(t, err)
	defer device.Close(ctx)

	// before own
	got, err := device.GetOwnership(ctx)
	require.NoError(t, err)
	assert.False(t, got.Owned)

	err = device.Own(ctx, links, c.otm)
	require.NoError(t, err)

	// after own
	got,err = device.GetOwnership(ctx)
	require.NoError(t, err)
	assert.True(t, got.Owned)
	assert.Equal(t, CertIdentity, got.DeviceOwner)

	err = device.Disown(ctx, links)
	require.NoError(t, err)
	
	// after disown
	got, err = device.GetOwnership(ctx)
	require.NoError(t, err)
	assert.False(t, got.Owned)
}
