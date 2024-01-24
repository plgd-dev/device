package bridge_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	bridgeTest "github.com/plgd-dev/device/v2/bridge/test"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	codecOcf "github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/stretchr/testify/require"
)

func TestObserveResource(t *testing.T) {
	s := bridgeTest.NewBridgeService(t)
	d := bridgeTest.NewBridgedDevice(t, s, false, uuid.New().String())
	defer func() {
		s.DeleteAndCloseDevice(d.GetID())
	}()

	rds := resourceDataSync{
		resourceData: resourceData{
			Name: "test",
		},
	}

	resHandler := func(req *net.Request) (*pool.Message, error) {
		resp := pool.NewMessage(req.Context())
		switch req.Code() {
		case codes.GET:
			resp.SetCode(codes.Content)
		case codes.POST:
			resp.SetCode(codes.Changed)
		default:
			return nil, fmt.Errorf("invalid method %v", req.Code())
		}
		resp.SetContentFormat(message.AppOcfCbor)
		data, err := cbor.Encode(rds.copy())
		if err != nil {
			return nil, err
		}
		resp.SetBody(bytes.NewReader(data))
		return resp, nil
	}

	res := resources.NewResource("/test", resHandler, func(req *net.Request) (*pool.Message, error) {
		codec := codecOcf.VNDOCFCBORCodec{}
		var newData resourceData
		err := codec.Decode(req.Message, &newData)
		if err != nil {
			return nil, err
		}
		rds.setName(newData.Name)
		return resHandler(req)
	}, []string{"oic.d.virtual", "oic.d.test"}, []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_RW})
	res.SetObserveHandler(func(req *net.Request, handler func(msg *pool.Message, err error)) (cancel func(), err error) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			defer cancel()
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Millisecond * 100):
					resp, err := resHandler(req)
					if err != nil {
						handler(nil, err)
						return
					}
					handler(resp, nil)
				}
			}
		}()
		return cancel, nil
	})
	d.AddResource(res)

	cleanup := bridgeTest.RunBridgeService(s)
	defer cleanup()

	c, err := testClient.NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	h := testClient.MakeMockResourceObservationHandler()
	obsID, err := c.ObserveResource(ctx, d.GetID().String(), "/test", h, withDeviceID(d.GetID().String()))
	require.NoError(t, err)
	defer func() {
		_, errC := c.StopObservingResource(ctx, obsID)
		require.NoError(t, errC)
	}()

	n, err := h.WaitForNotification(ctx)
	require.NoError(t, err)
	var originalResource resourceData
	err = n(&originalResource)
	require.NoError(t, err)
	require.Equal(t, "test", originalResource.Name)

	var got coap.DetailedResponse[interface{}]
	err = c.UpdateResource(ctx, d.GetID().String(), "/test", map[string]interface{}{
		"name": "updated",
	}, &got, withDeviceID(d.GetID().String()))
	require.NoError(t, err)
	require.Equal(t, codes.Changed, got.Code)
	require.Equal(t, "updated", rds.getName())

	n, err = h.WaitForNotification(ctx)
	require.NoError(t, err)
	var updatedResource resourceData
	err = n(&updatedResource)
	require.NoError(t, err)
	require.Equal(t, "updated", updatedResource.Name)

	// fail - invalid data
	err = c.UpdateResource(ctx, d.GetID().String(), "/test", map[string]interface{}{
		"name": 1,
	}, &got, withDeviceID(d.GetID().String()))
	require.Error(t, err)
}
