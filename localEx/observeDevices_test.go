package localEx_test

import (
	"context"
	"fmt"
	"runtime/debug"
	"testing"
	"time"

	"github.com/go-ocf/sdk/localEx"
	"github.com/go-ocf/sdk/test"

	"github.com/stretchr/testify/require"
)

func waitForDevicesObservationEvent(ctx context.Context, t *testing.T, chanDevs <-chan localEx.DevicesObservationEvent, expectedEvent localEx.DevicesObservationEvent) {
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
	defer func() {
		err := c.StopObservingDevices(ctx, ID)
		require.NoError(t, err)
	}()

	waitForDevicesObservationEvent(ctx, t, h.devs, localEx.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    localEx.DevicesObservationEvent_ONLINE,
	})

	/* TODO: add support for reboot to iotivity-lite
	err = c.Reboot(ctx, deviceID)
	require.NoError(t, err)

	waitForDevicesObservationEvent(ctx, t, h.devs, localEx.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    localEx.DevicesObservationEvent_OFFLINE,
	})
	waitForDevicesObservationEvent(ctx, t, h.devs, localEx.DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    localEx.DevicesObservationEvent_ONLINE,
	})
	*/
}

func makeTestDevicesObservationHandler() *testDevicesObservationHandler {
	return &testDevicesObservationHandler{devs: make(chan localEx.DevicesObservationEvent, 100)}
}

type testDevicesObservationHandler struct {
	devs chan localEx.DevicesObservationEvent
}

func (h *testDevicesObservationHandler) Handle(ctx context.Context, body localEx.DevicesObservationEvent) error {
	h.devs <- body
	return nil
}

func (h *testDevicesObservationHandler) Error(err error) {
	fmt.Println(err)
	debug.PrintStack()
}

func (h *testDevicesObservationHandler) OnClose() {
	fmt.Println("device resources observation was closed")
}
