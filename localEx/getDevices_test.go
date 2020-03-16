package localEx_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-ocf/sdk/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceDiscovery(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	secureDeviceID := test.MustFindDeviceByName(test.TestSecureDeviceName)
	h := func(err error) { fmt.Println(err) }
	c, err := NewTestSecureClient()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	devices, err := c.GetDevices(ctx, nil, h)
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	d := devices[deviceID]
	require.NotEmpty(t, d)
	assert.Equal(t, test.TestDeviceName, d.Device.Name)

	d = devices[secureDeviceID]
	fmt.Println(d)
	require.NotNil(t, d)
	assert.Equal(t, test.TestSecureDeviceName, d.Device.Name)
	require.NotNil(t, d.Ownership)
	assert.Equal(t, d.Ownership.DeviceOwner, "00000000-0000-0000-0000-000000000000")
}

func TestDeviceDiscoveryWithFilter(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.TestSecureDeviceName)
	h := func(err error) {}
	c := NewTestClient()
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	devices, err := c.GetDevices(ctx, []string{"oic.wk.d"}, h)
	require.NoError(t, err)
	assert.NotEmpty(t, devices[secureDeviceID], "unreachable test device")

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	devices, err = c.GetDevices(ctx, []string{"x.com.device"}, h)
	require.NoError(t, err)
	assert.Empty(t, devices, "test device not filtered out")
}
