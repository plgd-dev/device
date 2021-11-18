package client_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/plgd-dev/device/client"
	kitNetCoap "github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDevice(t *testing.T, name string, runTest func(t *testing.T, ctx context.Context, c *client.Client, deviceID string)) {
	deviceID := test.MustFindDeviceByName(name)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	deviceID, err = c.OwnDevice(ctx, deviceID)
	require.NoError(t, err)
	defer disown(t, c, deviceID)

	runTest(t, ctx, c, deviceID)
}

func runObservingResourceTest(t *testing.T, ctx context.Context, c *client.Client, deviceID string) {
	h := makeObservationHandler()
	id, err := c.ObserveResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), h)
	require.NoError(t, err)
	defer func(observationID string) {
		err := c.StopObservingResource(ctx, observationID)
		require.NoError(t, err)
	}(id)

	var d map[string]interface{}
	res := <-h.res
	err = res(&d)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d["power"].(uint64))

	h2 := makeObservationHandler()
	id, err = c.ObserveResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), h2)
	require.NoError(t, err)
	defer func(observationID string) {
		err := c.StopObservingResource(ctx, observationID)
		require.NoError(t, err)
	}(id)

	var d2 map[string]interface{}
	res = <-h2.res
	err = res(&d2)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d2["power"].(uint64))

	err = c.UpdateResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), map[string]interface{}{
		"power": uint64(123),
	}, nil)
	require.NoError(t, err)

	res = <-h.res
	err = res(&d)
	require.NoError(t, err)
	assert.Equal(t, uint64(123), d["power"].(uint64))

	res = <-h2.res
	err = res(&d2)
	require.NoError(t, err)
	assert.Equal(t, uint64(123), d2["power"].(uint64))

	err = c.UpdateResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), map[string]interface{}{
		"power": uint64(0),
	}, nil)
	assert.NoError(t, err)

	res = <-h.res
	err = res(&d)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d["power"].(uint64))

	res = <-h2.res
	err = res(&d2)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d2["power"].(uint64))
}

func TestObservingResource(t *testing.T) {
	testDevice(t, test.DevsimName, runObservingResourceTest)
}

func makeObservationHandler() *observationHandler {
	return &observationHandler{res: make(chan kitNetCoap.DecodeFunc, 1)}
}

type observationHandler struct {
	res chan kitNetCoap.DecodeFunc
}

func (h *observationHandler) Handle(ctx context.Context, body kitNetCoap.DecodeFunc) {
	h.res <- body
}

func (h *observationHandler) Error(err error) { fmt.Println(err) }

func (h *observationHandler) OnClose() { fmt.Println("Observation was closed") }
