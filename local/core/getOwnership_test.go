package core_test

import (
	"context"
	"testing"
	"time"

	grpcTest "github.com/go-ocf/grpc-gateway/test"
	"github.com/go-ocf/sdk/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOwnership(t *testing.T) {
	secureDeviceID := grpcTest.MustFindDeviceByName(test.TestSecureDeviceName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	deviceID := secureDeviceID
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
	got, err = device.GetOwnership(ctx)
	require.NoError(t, err)
	assert.True(t, got.Owned)
	assert.Equal(t, CertIdentity, got.DeviceOwner)

	err = device.Disown(ctx, links)
	require.NoError(t, err)
}