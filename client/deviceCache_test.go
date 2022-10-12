package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/schema"
	"github.com/stretchr/testify/require"
)

func TestDeviceCacheContentHandling(t *testing.T) {
	cache := client.NewDeviceCache(time.Second*5, time.Second, func(error) {})

	deviceType := []string{"unknown"}
	device1ID := "12345"
	device2ID := "abcd"

	newdev := core.NewDevice(core.DeviceConfiguration{}, device1ID, deviceType, func() schema.Endpoints { return nil })

	newdev2 := core.NewDevice(core.DeviceConfiguration{}, device2ID, deviceType, func() schema.Endpoints { return nil })

	dev, found := cache.GetDevice(device1ID)
	require.False(t, found)
	require.Nil(t, dev)

	dev, found = cache.GetDevice(device2ID)
	require.False(t, found)
	require.Nil(t, dev)

	dev, updated := cache.UpdateOrStoreDevice(newdev)
	require.False(t, updated)
	require.NotNil(t, dev)

	expiration, found := cache.GetDeviceExpiration(device1ID)
	require.True(t, found)
	require.True(t, expiration.IsZero())

	dev, found = cache.GetDevice(device1ID)
	require.True(t, found)
	require.NotNil(t, dev)

	dev, found = cache.GetDevice(device2ID)
	require.False(t, found)
	require.Nil(t, dev)

	dev, updated = cache.UpdateOrStoreDeviceWithExpiration(newdev2)
	require.False(t, updated)
	require.NotNil(t, dev)

	dev, found = cache.GetDevice(device2ID)
	require.True(t, found)
	require.NotNil(t, dev)

	expiration, found = cache.GetDeviceExpiration(device2ID)
	require.True(t, found)
	require.False(t, expiration.IsZero())

	dev, removed := cache.LoadAndDeleteDevice(device2ID)
	require.True(t, removed)
	err := dev.Close(context.TODO())
	require.NoError(t, err)
	dev, removed = cache.LoadAndDeleteDevice(device2ID)
	require.False(t, removed)
	require.Nil(t, dev)

	dev, found = cache.GetDevice(device1ID)
	require.True(t, found)
	require.NotNil(t, dev)

	ok := cache.TryToChangeDeviceExpirationToDefault(device1ID)
	require.True(t, ok)

	expiration, found = cache.GetDeviceExpiration(device1ID)
	require.True(t, found)
	require.False(t, expiration.IsZero())

	dev, removed = cache.LoadAndDeleteDevice(device1ID)
	require.True(t, removed)
	err = dev.Close(context.TODO())
	require.NoError(t, err)

	dev, found = cache.GetDevice(device1ID)
	require.False(t, found)
	require.Nil(t, dev)
}

func TestDeviceCacheExpirationHandling(t *testing.T) {
	expectedExpiration := time.Now().Add(5 * time.Second)
	cache := client.NewDeviceCache(5*time.Second, time.Second, func(error) {})

	deviceType := []string{"unknown"}
	deviceID := "12345"
	deviceID2 := "abcd"

	newdev := core.NewDevice(core.DeviceConfiguration{}, deviceID, deviceType, func() schema.Endpoints { return nil })

	cache.UpdateOrStoreDeviceWithExpiration(newdev)
	expiration, found := cache.GetDeviceExpiration(deviceID)
	require.True(t, found)
	require.LessOrEqual(t, expectedExpiration, expiration)

	// create the same device (the same deviceID) and try to store it without expiration
	// the device should stay in cache and just it's expiration should be updated
	newdev = core.NewDevice(core.DeviceConfiguration{}, deviceID, deviceType, func() schema.Endpoints { return nil })

	_, updated := cache.UpdateOrStoreDevice(newdev)
	require.True(t, updated)

	expiration, found = cache.GetDeviceExpiration(deviceID)
	require.True(t, found)
	require.True(t, expiration.IsZero())

	// create a second device and store it without expiration and the following
	// storage of the same device with a timeout should have no effect on the expiration
	newdev2 := core.NewDevice(core.DeviceConfiguration{}, deviceID2, deviceType, func() schema.Endpoints { return nil })

	_, updated = cache.UpdateOrStoreDevice(newdev2)
	require.False(t, updated)

	expiration, found = cache.GetDeviceExpiration(deviceID)
	require.True(t, found)
	require.True(t, expiration.IsZero())

	newdev2 = core.NewDevice(core.DeviceConfiguration{}, deviceID2, deviceType, func() schema.Endpoints { return nil })

	_, updated = cache.UpdateOrStoreDeviceWithExpiration(newdev2)
	require.True(t, updated)

	expiration, found = cache.GetDeviceExpiration(deviceID2)
	require.True(t, found)
	require.True(t, expiration.IsZero())

	ok := cache.TryToChangeDeviceExpirationToDefault(deviceID2)
	require.True(t, ok)

	time.Sleep(6 * time.Second)

	// the device with deviceID should be in the cache because is stored for infinite time
	expiration, found = cache.GetDeviceExpiration(deviceID)
	require.True(t, found)
	require.True(t, expiration.IsZero())

	// the device with deviceID2 should be removed from the cache by the expiration
	_, found = cache.GetDevice(deviceID2)
	require.False(t, found)
}

func TestDeviceCacheExpirationWithInfiniteExpiration(t *testing.T) {
	cache := client.NewDeviceCache(0, time.Second, func(error) {})
	deviceType := []string{"unknown"}
	deviceID := "12345"
	newdev := core.NewDevice(core.DeviceConfiguration{}, deviceID, deviceType, func() schema.Endpoints { return nil })
	d, updated := cache.UpdateOrStoreDeviceWithExpiration(newdev)
	require.False(t, updated)
	require.NotNil(t, d)
	expiration, found := cache.GetDeviceExpiration(deviceID)
	require.True(t, found)
	require.True(t, expiration.IsZero())
	d, updated = cache.UpdateOrStoreDevice(newdev)
	require.True(t, updated)
	require.NotNil(t, d)
	expiration, found = cache.GetDeviceExpiration(deviceID)
	require.True(t, found)
	require.True(t, expiration.IsZero())
}
