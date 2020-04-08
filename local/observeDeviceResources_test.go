package local_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	grpcTest "github.com/go-ocf/cloud/grpc-gateway/test"
	"github.com/go-ocf/sdk/local"
	"github.com/go-ocf/sdk/schema"

	"github.com/stretchr/testify/require"
)

func TestObserveDeviceResources(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(TestDeviceName)
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
	defer func() {
		c.StopObservingDeviceResources(ctx, ID)
	}()

LOOP:
	for {
		select {
		case res := <-h.res:
			if res.Link.Href == "/oic/d" {
				res.Link.Endpoints = nil
				require.Equal(t, local.DeviceResourcesObservationEvent{
					Link: schema.ResourceLink{
						Href:          "/oic/d",
						ResourceTypes: []string{"oic.d.cloudDevice", "oic.wk.d"},
						Interfaces:    []string{"oic.if.r", "oic.if.baseline"},
						Anchor:        "ocf://" + deviceID,
						Policy: schema.Policy{
							BitMask: schema.Discoverable,
						},
					},
					Event: local.DeviceResourcesObservationEvent_ADDED,
				}, res)
				break LOOP
			}
		case <-ctx.Done():
			require.NoError(t, fmt.Errorf("timeout"))
			break LOOP
		}
	}
}

func makeTestDeviceResourcesObservationHandler() *testDeviceResourcesObservationHandler {
	return &testDeviceResourcesObservationHandler{res: make(chan local.DeviceResourcesObservationEvent, 100)}
}

type testDeviceResourcesObservationHandler struct {
	res chan local.DeviceResourcesObservationEvent
}

func (h *testDeviceResourcesObservationHandler) Handle(ctx context.Context, body local.DeviceResourcesObservationEvent) error {
	h.res <- body
	return nil
}

func (h *testDeviceResourcesObservationHandler) Error(err error) {
	fmt.Println(err)
}

func (h *testDeviceResourcesObservationHandler) OnClose() {
	fmt.Println("device resources observation was closed")
}
