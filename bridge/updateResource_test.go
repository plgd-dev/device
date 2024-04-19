package bridge_test

import (
	"bytes"
	"context"
	"sync"
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

type resourceData struct {
	Name string `json:"name,omitempty"`
}

type resourceDataSync struct {
	resourceData
	lock sync.Mutex
}

func (r *resourceDataSync) setName(name string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.Name = name
}

func (r *resourceDataSync) getName() string {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.Name
}

func (r *resourceDataSync) copy() resourceData {
	r.lock.Lock()
	defer r.lock.Unlock()
	return resourceData{
		Name: r.Name,
	}
}

func TestUpdateResource(t *testing.T) {
	s := bridgeTest.NewBridgeService(t)
	t.Cleanup(func() {
		_ = s.Shutdown()
	})
	d := bridgeTest.NewBridgedDevice(t, s, uuid.New().String(), false, false, false)
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
		data, err := cbor.Encode(rds.copy())
		if err != nil {
			return nil, err
		}
		resp.SetCode(codes.Changed)
		resp.SetContentFormat(message.AppOcfCbor)
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
	d.AddResources(res)

	cleanup := bridgeTest.RunBridgeService(s)
	defer func() {
		errC := cleanup()
		require.NoError(t, errC)
	}()

	c, err := testClient.NewTestSecureClientWithBridgeSupport()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()
	var got coap.DetailedResponse[interface{}]
	err = c.UpdateResource(ctx, d.GetID().String(), "/test", map[string]interface{}{
		"name": "updated",
	}, &got)
	require.NoError(t, err)
	require.Equal(t, codes.Changed, got.Code)
	require.Equal(t, "updated", rds.getName())

	// fail - invalid data
	err = c.UpdateResource(ctx, d.GetID().String(), "/test", map[string]interface{}{
		"name": 1,
	}, &got)
	require.Error(t, err)

	// fail - invalid href
	err = c.UpdateResource(ctx, d.GetID().String(), "/invalid", map[string]interface{}{
		"name": "updated",
	}, &got)
	require.Error(t, err)
}
