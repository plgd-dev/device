package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/client"
	kitNetCoap "github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/cloud"
	"github.com/plgd-dev/device/schema/configuration"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/schema/introspection"
	"github.com/plgd-dev/device/schema/maintenance"
	"github.com/plgd-dev/device/schema/platform"
	"github.com/plgd-dev/device/schema/resources"
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
	res, err := h.waitForNotification(ctx)
	require.NoError(t, err)
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
	res, err = h2.waitForNotification(ctx)
	require.NoError(t, err)
	err = res(&d2)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d2["power"].(uint64))

	err = c.UpdateResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), map[string]interface{}{
		"power": uint64(123),
	}, nil)
	require.NoError(t, err)

	res, err = h.waitForNotification(ctx)
	require.NoError(t, err)
	err = res(&d)
	require.NoError(t, err)
	assert.Equal(t, uint64(123), d["power"].(uint64))

	res, err = h2.waitForNotification(ctx)
	require.NoError(t, err)
	err = res(&d2)
	require.NoError(t, err)
	assert.Equal(t, uint64(123), d2["power"].(uint64))

	err = c.UpdateResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), map[string]interface{}{
		"power": uint64(0),
	}, nil)
	assert.NoError(t, err)

	res, err = h.waitForNotification(ctx)
	require.NoError(t, err)
	err = res(&d)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d["power"].(uint64))

	res, err = h2.waitForNotification(ctx)
	require.NoError(t, err)
	err = res(&d2)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d2["power"].(uint64))
}

func makeObservationHandler() *observationHandler {
	return &observationHandler{res: make(chan kitNetCoap.DecodeFunc, 1), close: make(chan struct{})}
}

type observationHandler struct {
	res   chan kitNetCoap.DecodeFunc
	close chan struct{}
}

func (h *observationHandler) Handle(ctx context.Context, body kitNetCoap.DecodeFunc) {
	h.res <- body
}

func (h *observationHandler) Error(err error) { fmt.Println(err) }

func (h *observationHandler) OnClose() { close(h.close) }

func (h *observationHandler) waitForNotification(ctx context.Context) (body kitNetCoap.DecodeFunc, error error) {
	select {
	case e := <-h.res:
		return e, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.close:
		return nil, fmt.Errorf("unexpected close")
	}
}

func (h *observationHandler) waitForClose(ctx context.Context) error {
	select {
	case e := <-h.res:
		var d interface{}
		e(d)
		return fmt.Errorf("unexpected notification %v", d)
	case <-ctx.Done():
		return ctx.Err()
	case <-h.close:
		return nil
	}
}

func TestObservingResource(t *testing.T) {
	testDevice(t, test.DevsimName, runObservingResourceTest)
}

func TestObservingDiscoveryResource(t *testing.T) {
	testDevice(t, test.DevsimName, func(t *testing.T, ctx context.Context, c *client.Client, deviceID string) {
		h := makeObservationHandler()
		id, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h)
		require.NoError(t, err)
		var d schema.ResourceLinks
		res, err := h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d)
		require.NoError(t, err)
		assert.NotEmpty(t, d)
		numResources := len(d)
		d.Sort()
		checkForNonObservationCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		err = h.waitForClose(checkForNonObservationCtx)
		if err == nil {
			// oic/res doesn't support observation
			return
		}
		defer func(observationID string) {
			err = c.StopObservingResource(ctx, observationID)
			require.NoError(t, err)
		}(id)
		require.Equal(t, context.DeadlineExceeded, err)
		createSwitch(ctx, t, c, deviceID)
		var d1 schema.ResourceLinks
		res, err = h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d1)
		require.NoError(t, err)
		d1.Sort()
		require.Equal(t, numResources+1, len(d1))
		var j int
		for i := range d1 {
			if j < len(d) && d[j].Href == d1[i].Href {
				require.Equal(t, d[j], d1[i])
				j++
			} else {
				require.Equal(t, test.TestResourceSwitchesInstanceHref("1"), d1[i].Href)
			}
		}
		err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref("1"), nil)
		require.NoError(t, err)
		var d2 schema.ResourceLinks
		res, err = h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d2)
		require.NoError(t, err)
		d2.Sort()
		require.Equal(t, d, d2)
	})
}

func TestObservingDiscoveryResourceWithBaselineInterface(t *testing.T) {
	testDevice(t, test.DevsimName, func(t *testing.T, ctx context.Context, c *client.Client, deviceID string) {
		h := makeObservationHandler()
		id, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_BASELINE))
		require.NoError(t, err)
		var d resources.BaselineResourceDiscovery
		res, err := h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d)
		require.NoError(t, err)
		require.NotEmpty(t, d)
		numResources := len(d[0].Links)
		d[0].Links.Sort()
		checkForNonObservationCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		err = h.waitForClose(checkForNonObservationCtx)
		if err == nil {
			// oic/res doesn't support observation
			return
		}
		defer func(observationID string) {
			err = c.StopObservingResource(ctx, observationID)
			require.NoError(t, err)
		}(id)
		require.Equal(t, context.DeadlineExceeded, err)
		createSwitch(ctx, t, c, deviceID)
		var d1 resources.BaselineResourceDiscovery
		res, err = h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d1)
		require.NoError(t, err)
		require.NotEmpty(t, d1)
		d1[0].Links.Sort()
		require.Equal(t, numResources+1, len(d1[0].Links))
		var j int
		for i := range d1 {
			if j < len(d) && d[0].Links[j].Href == d1[0].Links[i].Href {
				require.Equal(t, d[0].Links[j], d1[0].Links[i])
				j++
			} else {
				require.Equal(t, test.TestResourceSwitchesInstanceHref("1"), d1[0].Links[i].Href)
			}
		}
		err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref("1"), nil)
		require.NoError(t, err)
		var d2 resources.BaselineResourceDiscovery
		res, err = h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d2)
		require.NoError(t, err)
		require.NotEmpty(t, d1)
		d2[0].Links.Sort()
		require.Equal(t, d, d2)
	})
}

func TestObservingNonObservableResource(t *testing.T) {
	testDevice(t, test.DevsimName, func(t *testing.T, ctx context.Context, c *client.Client, deviceID string) {
		h := makeObservationHandler()
		_, err := c.ObserveResource(ctx, deviceID, maintenance.ResourceURI, h)
		require.NoError(t, err)
		var d maintenance.Maintenance
		res, err := h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d)
		require.NoError(t, err)
		err = h.waitForClose(ctx)
		require.NoError(t, err)
	})
}

func TestObservingDiscoveryResourceWithBatchInterface(t *testing.T) {
	testDevice(t, test.DevsimName, func(t *testing.T, ctx context.Context, c *client.Client, deviceID string) {
		h := makeObservationHandler()
		id, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B))
		require.NoError(t, err)
		var d resources.BatchResourceDiscovery
		res, err := h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d)
		require.NoError(t, err)
		assert.NotEmpty(t, d)
		d.Sort()
		require.Len(t, d, 8)
		for i := range d {
			require.Equal(t, deviceID, d[i].DeviceID())
			switch d[i].Href() {
			case device.ResourceURI:
			case platform.ResourceURI:
			case test.TestResourceLightInstanceHref("1"):
			case cloud.ConfigurationResourceURI:
			case maintenance.ResourceURI:
			case introspection.ResourceURI:
			case configuration.ResourceURI:
			case test.TestResourceSwitchesHref:
			default:
				require.NoError(t, fmt.Errorf("unknown resource href: %v", d[i].Href()))
			}
		}
		checkForNonObservationCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		err = h.waitForClose(checkForNonObservationCtx)
		if err == nil {
			// oic/res doesn't support observation
			return
		}
		defer func(observationID string) {
			err := c.StopObservingResource(ctx, observationID)
			require.NoError(t, err)
		}(id)
		require.Equal(t, context.DeadlineExceeded, err)
		createSwitch(ctx, t, c, deviceID)
		var d1 resources.BatchResourceDiscovery
		res, err = h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d1)
		require.NoError(t, err)
		d1.Sort()
		require.Len(t, d1, 2)
		for i := range d1 {
			require.Equal(t, deviceID, d1[i].DeviceID())
			switch d1[i].Href() {
			case test.TestResourceSwitchesInstanceHref("1"):
			case test.TestResourceSwitchesHref:
			default:
				require.NoError(t, fmt.Errorf("unknown resource href: %v", d1[i].Href()))
			}
		}
		err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref("1"), nil)
		require.NoError(t, err)
		var d2 resources.BatchResourceDiscovery
		res, err = h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d2)
		require.NoError(t, err)
		d2.Sort()
		require.Len(t, d2, 1)
		for i := range d2 {
			require.Equal(t, deviceID, d2[i].DeviceID())
			switch d2[i].Href() {
			case test.TestResourceSwitchesHref:
			default:
				require.NoError(t, fmt.Errorf("unknown resource href: %v", d2[i].Href()))
			}
		}
	})
}