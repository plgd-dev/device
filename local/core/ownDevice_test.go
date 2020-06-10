package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-ocf/sdk/test"
	"github.com/stretchr/testify/require"
)

func TestClient_ownDevice(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.TestSecureDeviceName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()
	deviceId := secureDeviceID
	require := require.New(t)
	timeout, cancelTimeout := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelTimeout()

	device, links, err := c.GetDevice(timeout, deviceId)
	require.NoError(err)
	defer device.Close(timeout)

	err = device.Own(timeout, links, c.otm)
	require.NoError(err)
	err = device.Disown(timeout, links)
	require.NoError(err)

	// try disown second time
	secureDeviceID = test.MustFindDeviceByName(test.TestSecureDeviceName)
	device, links, err = c.GetDevice(timeout, deviceId)
	require.NoError(err)
	defer device.Close(timeout)
	err = device.Disown(timeout, links)
	require.NoError(err)
}
