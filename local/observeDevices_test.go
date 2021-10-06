package local_test

import (
	"context"
	"fmt"
	"runtime/debug"
	"testing"
	"time"

	"github.com/plgd-dev/sdk/v2/local"
	"github.com/plgd-dev/sdk/v2/test"

	"github.com/stretchr/testify/require"
)

func waitForDevicesObservationEvent(ctx context.Context, t *testing.T, chanDevs <-chan local.DevicesObservationEvent, expectedEvent local.DevicesObservationEvent) {
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

func TestObserveDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	c := NewTestClient()
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	h := makeTestDevicesObservationHandler()
	ID, err := c.ObserveDevices(ctx, h)
	require.NoError(t, err)

	waitForDevicesObservationEvent(ctx, t, h.devs, local.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    local.DevicesObservationEvent_ONLINE,
	})
	/* TODO: add support for reboot to iotivity-lite
	err = c.Reboot(ctx, deviceID)
	require.NoError(t, err)

	waitForDevicesObservationEvent(ctx, t, h.devs, local.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    local.DevicesObservationEvent_OFFLINE,
	})
	waitForDevicesObservationEvent(ctx, t, h.devs, local.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    local.DevicesObservationEvent_ONLINE,
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

	err = c.StopObservingDevices(ctx, ID)
	require.NoError(t, err)
	select {
	case <-h.devs:
		require.NoError(t, fmt.Errorf("unexpected event"))
	default:
	}
}

func makeTestDevicesObservationHandler() *testDevicesObservationHandler {
	return &testDevicesObservationHandler{devs: make(chan local.DevicesObservationEvent, 100)}
}

type testDevicesObservationHandler struct {
	devs chan local.DevicesObservationEvent
}

func (h *testDevicesObservationHandler) Handle(ctx context.Context, body local.DevicesObservationEvent) error {
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
