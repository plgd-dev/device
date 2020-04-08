package backend_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-ocf/sdk/backend"

	authTest "github.com/go-ocf/cloud/authorization/provider"
	"github.com/go-ocf/cloud/grpc-gateway/pb"
	grpcTest "github.com/go-ocf/cloud/grpc-gateway/test"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/stretchr/testify/require"
)

func TestObserveDeviceResources(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)

	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	c := NewTestClient(t)
	defer c.Close(context.Background())
	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

	h := makeTestDeviceResourcesObservationHandler()
	id, err := c.ObserveDeviceResources(ctx, deviceID, h)
	require.NoError(t, err)
	defer func() {
		err := c.StopObservingDevices(ctx, id)
		require.NoError(t, err)
	}()

LOOP:
	for {
		select {
		case res := <-h.res:
			t.Logf("res %+v\n", res)
			if res.Link.GetHref() == "/oic/d" {
				require.Equal(t, backend.DeviceResourcesObservationEvent{
					Link: pb.ResourceLink{
						Href:       "/oic/d",
						Types:      []string{"oic.d.cloudDevice", "oic.wk.d"},
						Interfaces: []string{"oic.if.r", "oic.if.baseline"},
						DeviceId:   deviceID,
					},
					Event: backend.DeviceResourcesObservationEvent_ADDED,
				}, res)
				break LOOP
			}
		case <-time.After(TestTimeout):
			t.Error("timeout")
			break LOOP
		}

	}
}

func makeTestDeviceResourcesObservationHandler() *testDeviceResourcesObservationHandler {
	return &testDeviceResourcesObservationHandler{res: make(chan backend.DeviceResourcesObservationEvent, 100)}
}

type testDeviceResourcesObservationHandler struct {
	res chan backend.DeviceResourcesObservationEvent
}

func (h *testDeviceResourcesObservationHandler) Handle(ctx context.Context, body backend.DeviceResourcesObservationEvent) error {
	h.res <- body
	return nil
}

func (h *testDeviceResourcesObservationHandler) Error(err error) { fmt.Println(err) }

func (h *testDeviceResourcesObservationHandler) OnClose() {
	fmt.Println("devices observation was closed")
}
