package client_test

import (
	"context"
	"fmt"
	"runtime/debug"
	"testing"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/require"
)

func waitForDevicesObservationEvent(ctx context.Context, t *testing.T, chanDevs <-chan client.DevicesObservationEvent, expectedEvent client.DevicesObservationEvent) {
LOOP:
	for {
		select {
		case devs := <-chanDevs:
			if devs.DeviceID == expectedEvent.DeviceID {
				require.Equal(t, expectedEvent, devs)
				break LOOP
			}
		case <-ctx.Done():
			require.NoError(t, fmt.Errorf("timeout"))
			break LOOP
		}
	}
}

func TestObserveDevicesAddedByIP(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	ip := test.MustFindDeviceIP(test.DevsimName, test.IP4)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout*2)
	defer cancel()

	h := makeTestDevicesObservationHandler()
	discoveryConfig := core.DiscoveryConfiguration{}
	ID, err := c.ObserveDevices(ctx, h, client.WithDiscoveryConfiguration(discoveryConfig))
	require.NoError(t, err)
	_, err = c.GetDeviceByIP(ctx, ip)
	require.NoError(t, err)

	waitForDevicesObservationEvent(ctx, t, h.devs, client.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    client.DevicesObservationEvent_ONLINE,
	})

	ok, err := c.RemoveDevice(ctx, deviceID)
	require.NoError(t, err)
	require.True(t, ok)

	waitForDevicesObservationEvent(ctx, t, h.devs, client.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    client.DevicesObservationEvent_OFFLINE,
	})

LOOP:
	for {
		select {
		case <-h.devs:
		default:
			break LOOP
		}
	}

	ok = c.StopObservingDevices(ctx, ID)
	require.True(t, ok)
	select {
	case <-h.devs:
		require.NoError(t, fmt.Errorf("unexpected event"))
	default:
	}
}

func TestObserveDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout*2)
	defer cancel()

	h := makeTestDevicesObservationHandler()
	ID, err := c.ObserveDevices(ctx, h)
	require.NoError(t, err)

	waitForDevicesObservationEvent(ctx, t, h.devs, client.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    client.DevicesObservationEvent_ONLINE,
	})
	/* TODO: add support for reboot to iotivity-lite
	err = c.Reboot(ctx, deviceID)
	require.NoError(t, err)

	waitForDevicesObservationEvent(ctx, t, h.devs, client.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    client.DevicesObservationEvent_OFFLINE,
	})
	waitForDevicesObservationEvent(ctx, t, h.devs, client.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    client.DevicesObservationEvent_ONLINE,
	})
	*/
LOOP:
	for {
		select {
		case <-h.devs:
		default:
			break LOOP
		}
	}

	ok := c.StopObservingDevices(ctx, ID)
	require.True(t, ok)
	select {
	case <-h.devs:
		require.NoError(t, fmt.Errorf("unexpected event"))
	default:
	}
}

func makeTestDevicesObservationHandler() *testDevicesObservationHandler {
	return &testDevicesObservationHandler{devs: make(chan client.DevicesObservationEvent, 100)}
}

type testDevicesObservationHandler struct {
	devs chan client.DevicesObservationEvent
}

func (h *testDevicesObservationHandler) Handle(ctx context.Context, body client.DevicesObservationEvent) error {
	h.devs <- body
	return nil
}

func (h *testDevicesObservationHandler) Error(err error) {
	fmt.Println(err)
	debug.PrintStack()
}

func (h *testDevicesObservationHandler) OnClose() {
	fmt.Println("devices observation was closed")
}
