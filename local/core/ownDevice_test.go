package core_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/kit/net/coap"
	"github.com/plgd-dev/sdk/local/core"
	"github.com/plgd-dev/sdk/schema"
	"github.com/plgd-dev/sdk/test"
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
	device, links, err = c.GetDevice(timeout, secureDeviceID)
	require.NoError(err)
	defer device.Close(timeout)
	err = device.Disown(timeout, links)
	require.NoError(err)

	secureDeviceID = test.MustFindDeviceByName(test.TestSecureDeviceName)
	device, links, err = c.GetDevice(timeout, secureDeviceID)
	var newDeviceID string
	err = device.Own(timeout, links, c.otm, core.WithActionDuringOwn(func(ctx context.Context, client *coap.ClientCloseHandler) error {
		var d schema.Device
		err := client.GetResource(ctx, "/oic/d", &d)
		if err != nil {
			return core.MakeInternal(fmt.Errorf("cannot get device resource for owned device(%v): %w", secureDeviceID, err))
		}
		setDeviceOwned := schema.DoxmUpdate{
			DeviceID: d.ProtocolIndependentID,
		}
		/*doxm doesn't send any content for select OTM*/
		err = client.UpdateResource(ctx, "/oic/sec/doxm", setDeviceOwned, nil)
		if err != nil {
			return core.MakeInternal(fmt.Errorf("cannot set device id %v for owned device(%v): %w", d.ProtocolIndependentID, secureDeviceID, err))
		}
		newDeviceID = d.ProtocolIndependentID
		return nil
	}))
	require.NoError(err)
	require.NotEqual(t, secureDeviceID, newDeviceID)
	device, _, err = c.GetDevice(timeout, newDeviceID)
	require.NoError(err)
	err = device.Disown(timeout, links)
	require.NoError(err)
}
