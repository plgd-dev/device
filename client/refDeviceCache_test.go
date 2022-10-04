package client_test

import (
	"testing"
	"time"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/schema"
	"github.com/stretchr/testify/require"
)

func TestDeviceCacheContentHandling(t *testing.T) {
	cache := client.NewRefDeviceCache(time.Minute, func(error) {})

	deviceType := []string{"unknown"}
	device1ID := "12345"
	device2ID := "abcd"

	dev := core.NewDevice(core.DeviceConfiguration{}, device1ID, deviceType, func() schema.Endpoints { return nil })
	newRefDev := client.NewRefDevice(dev)

	dev2 := core.NewDevice(core.DeviceConfiguration{}, device2ID, deviceType, func() schema.Endpoints { return nil })
	newRefDev2 := client.NewRefDevice(dev2)

	refDev, found := cache.GetDevice(device1ID)
	require.False(t, found)
	require.Nil(t, refDev)

	refDev, found = cache.GetDevice(device2ID)
	require.False(t, found)
	require.Nil(t, refDev)

	refDev, stored := cache.TryStoreDevice(newRefDev)
	require.True(t, stored)
	require.NotNil(t, refDev)

	refDev, found = cache.GetDevice(device1ID)
	require.True(t, found)
	require.NotNil(t, refDev)

	refDev, found = cache.GetDevice(device2ID)
	require.False(t, found)
	require.Nil(t, refDev)

	refDev, stored = cache.TryStoreDevice(newRefDev2)
	require.True(t, stored)
	require.NotNil(t, refDev)

	refDev, found = cache.GetDevice(device2ID)
	require.True(t, found)
	require.NotNil(t, refDev)

	removed := cache.RemoveDevice(device2ID, refDev)
	require.True(t, removed)
	removed = cache.RemoveDevice(device2ID, refDev)
	require.False(t, removed)

	refDev, found = cache.GetDevice(device1ID)
	require.True(t, found)
	require.NotNil(t, refDev)

	removed = cache.RemoveDevice(device1ID, refDev)
	require.True(t, removed)

	refDev, found = cache.GetDevice(device1ID)
	require.False(t, found)
	require.Nil(t, refDev)
}

func TestDeviceCacheExpirationHandling(t *testing.T) {
	expectedExpiration := time.Now().Add(time.Minute)
	cache := client.NewRefDeviceCache(time.Minute, func(error) {})

	deviceType := []string{"unknown"}
	deviceID := "12345"
	deviceID2 := "abcd"

	dev := core.NewDevice(core.DeviceConfiguration{}, deviceID, deviceType, func() schema.Endpoints { return nil })
	newRefDev := client.NewRefDevice(dev)

	cache.TryStoreDevice(newRefDev)
	expiration, found := cache.GetDeviceExpiration(deviceID)
	require.True(t, found)
	require.LessOrEqual(t, expectedExpiration, expiration)

	// create the same device (the same deviceID) and try to store it without expiration
	// the device should stay in cache and just it's expiration should be updated
	dev = core.NewDevice(core.DeviceConfiguration{}, deviceID, deviceType, func() schema.Endpoints { return nil })
	newRefDev = client.NewRefDevice(dev)

	_, stored := cache.TryStoreDeviceWithoutTimeout(newRefDev)
	require.False(t, stored)

	expiration, found = cache.GetDeviceExpiration(deviceID)
	require.True(t, found)
	require.True(t, expiration.IsZero())

	// create a second device and store it without expiration and the following
	// storage of the same device with a timeout should have no effect on the expiration
	dev2 := core.NewDevice(core.DeviceConfiguration{}, deviceID2, deviceType, func() schema.Endpoints { return nil })
	newRefDev2 := client.NewRefDevice(dev2)

	_, stored = cache.TryStoreDeviceWithoutTimeout(newRefDev2)
	require.True(t, stored)

	expiration, found = cache.GetDeviceExpiration(deviceID)
	require.True(t, found)
	require.True(t, expiration.IsZero())

	dev2 = core.NewDevice(core.DeviceConfiguration{}, deviceID2, deviceType, func() schema.Endpoints { return nil })
	newRefDev2 = client.NewRefDevice(dev2)

	_, stored = cache.TryStoreDevice(newRefDev2)
	require.False(t, stored)

	expiration, found = cache.GetDeviceExpiration(deviceID)
	require.True(t, found)
	require.True(t, expiration.IsZero())
}
