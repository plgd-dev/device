package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceDiscovery(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	devices, err := c.GetDevices(ctx)
	require.NoError(t, err)

	d := devices[deviceID]
	require.NotEmpty(t, d)
	assert.Equal(t, test.DevsimName, d.Details.(*device.Device).Name)

	d = devices[secureDeviceID]
	fmt.Println(d)
	require.NotNil(t, d)
	assert.Equal(t, test.DevsimName, d.Details.(*device.Device).Name)
	require.NotNil(t, d.Ownership)
	assert.Equal(t, d.Ownership.OwnerID, "00000000-0000-0000-0000-000000000000")

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	secureDeviceID, err = c.OwnDevice(ctx, secureDeviceID, client.WithOTMs([]client.OTMType{client.OTMType_JustWorks}))
	require.NoError(t, err)
	devices, err = c.GetDevices(ctx)
	require.NoError(t, err)

	d = devices[secureDeviceID]
	fmt.Println(d)
	require.NotNil(t, d)
	require.NotNil(t, d.Ownership)
	sdkID, err := c.CoreClient().GetSdkOwnerID()
	require.NoError(t, err)
	assert.Equal(t, d.Ownership.OwnerID, sdkID)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = c.DisownDevice(ctx, secureDeviceID)
	require.NoError(t, err)
}

func TestDeviceDiscoveryWithFilter(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	devices, err := c.GetDevices(ctx, client.WithResourceTypes(device.ResourceType))
	require.NoError(t, err)
	assert.NotEmpty(t, devices[secureDeviceID], "unreachable test device")

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	devices, err = c.GetDevices(ctx, client.WithResourceTypes("x.com.device"))
	require.NoError(t, err)
	assert.Empty(t, devices, "test device not filtered out")
}

func TestDevicesWithFoundByIP(t *testing.T) {
	ip4 := test.MustFindDeviceIP(test.DevsimName, test.IP4)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	dev, err := c.GetDeviceByIP(ctx, ip4)
	require.NoError(t, err)
	require.NotNil(t, dev)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	devices, err := c.GetDevices(ctx, client.WithDiscoveryConfiguration(core.DiscoveryConfiguration{}))
	require.NoError(t, err)
	assert.Equal(t, devices[dev.ID], dev)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	devices, err = c.GetDevices(ctx, client.WithDiscoveryConfiguration(core.DiscoveryConfiguration{}), client.WithResourceTypes(device.ResourceType))
	require.NoError(t, err)
	assert.Equal(t, devices[dev.ID], dev)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	devices, err = c.GetDevices(ctx, client.WithDiscoveryConfiguration(core.DiscoveryConfiguration{}), client.WithResourceTypes("x.com.device"))
	require.NoError(t, err)
	assert.Empty(t, devices, "test device not filtered out")

	require.True(t, c.DeleteDevice(ctx, dev.ID))
	devices, err = c.GetDevices(ctx, client.WithDiscoveryConfiguration(core.DiscoveryConfiguration{}), client.WithResourceTypes(device.ResourceType))
	require.NoError(t, err)
	assert.Empty(t, devices, "empty devices are expected")
}
