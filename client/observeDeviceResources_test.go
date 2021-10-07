package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/test"

	"github.com/stretchr/testify/require"
)

func TestObserveDeviceResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	c := NewTestClient()
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	h := makeTestDeviceResourcesObservationHandler()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	ID, err := c.ObserveDeviceResources(ctx, deviceID, h)
	require.NoError(t, err)

LOOP:
	for {
		select {
		case res := <-h.res:
			if res.Link.Href == "/oic/d" {
				res.Link.Endpoints = nil
				require.Equal(t, client.DeviceResourcesObservationEvent{
					Link: schema.ResourceLink{
						Href:          "/oic/d",
						ResourceTypes: []string{"oic.d.cloudDevice", "oic.wk.d"},
						Interfaces:    []string{"oic.if.r", "oic.if.baseline"},
						Anchor:        "ocf://" + deviceID,
						Policy: &schema.Policy{
							BitMask: schema.Discoverable | schema.Observable,
						},
					},
					Event: client.DeviceResourcesObservationEvent_ADDED,
				}, res)
				break LOOP
			}
		case <-ctx.Done():
			require.NoError(t, fmt.Errorf("timeout"))
			break LOOP
		}
	}

LOOP1:
	for {
		select {
		case <-h.res:
		default:
			break LOOP1
		}
	}

	err = c.StopObservingDeviceResources(ctx, ID)
	require.NoError(t, err)
	select {
	case <-h.res:
		require.NoError(t, fmt.Errorf("unexpected event"))
	default:
	}
}

func makeTestDeviceResourcesObservationHandler() *testDeviceResourcesObservationHandler {
	return &testDeviceResourcesObservationHandler{res: make(chan client.DeviceResourcesObservationEvent, 100)}
}

type testDeviceResourcesObservationHandler struct {
	res chan client.DeviceResourcesObservationEvent
}

func (h *testDeviceResourcesObservationHandler) Handle(ctx context.Context, body client.DeviceResourcesObservationEvent) error {
	h.res <- body
	return nil
}

func (h *testDeviceResourcesObservationHandler) Error(err error) {
	fmt.Println(err)
}

func (h *testDeviceResourcesObservationHandler) OnClose() {
	fmt.Println("device resources observation was closed")
}
