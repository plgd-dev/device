// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/introspection"
	"github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeObservationHandler() *observationHandler {
	return &observationHandler{res: make(chan coap.DecodeFunc, 1), close: make(chan struct{})}
}

type observationHandler struct {
	res   chan coap.DecodeFunc
	close chan struct{}
}

func (h *observationHandler) Handle(_ context.Context, body coap.DecodeFunc) {
	h.res <- body
}

func (h *observationHandler) Error(err error) { fmt.Println(err) }

func (h *observationHandler) OnClose() { close(h.close) }

func (h *observationHandler) waitForNotification(ctx context.Context) (coap.DecodeFunc, error) {
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
		if err := e(d); err != nil {
			return fmt.Errorf("unexpected notification: cannot decode: %w", err)
		}
		return fmt.Errorf("unexpected notification %v", d)
	case <-ctx.Done():
		return ctx.Err()
	case <-h.close:
		return nil
	}
}

func testDevice(t *testing.T, name string, runTest func(ctx context.Context, t *testing.T, c *client.Client, deviceID string)) {
	deviceID := test.MustFindDeviceByName(name)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	deviceID, err = c.OwnDevice(ctx, deviceID)
	require.NoError(t, err)
	defer disown(t, c, deviceID)

	runTest(ctx, t, c, deviceID)
}

func runObservingResourceTest(ctx context.Context, t *testing.T, c *client.Client, deviceID string) {
	h := makeObservationHandler()
	id, err := c.ObserveResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), h)
	require.NoError(t, err)
	defer func(observationID string) {
		_, errC := c.StopObservingResource(ctx, observationID)
		require.NoError(t, errC)
	}(id)

	res, err := h.waitForNotification(ctx)
	require.NoError(t, err)
	d := coap.DetailedResponse[map[string]interface{}]{}
	err = res(&d)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d.Body["power"].(uint64))
	etag1 := d.ETag
	if ETagSupported {
		assert.NotEmpty(t, d.ETag)
	}

	h2 := makeObservationHandler()
	id, err = c.ObserveResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), h2)
	require.NoError(t, err)
	defer func(observationID string) {
		_, errC := c.StopObservingResource(ctx, observationID)
		require.NoError(t, errC)
	}(id)

	res, err = h2.waitForNotification(ctx)
	require.NoError(t, err)
	d2 := coap.DetailedResponse[map[string]interface{}]{}
	err = res(&d2)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d2.Body["power"].(uint64))
	etag2 := d2.ETag
	if ETagSupported {
		assert.NotEmpty(t, d2.ETag)
		require.Equal(t, d.ETag, d2.ETag)
	}

	err = c.UpdateResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), map[string]interface{}{
		"power": uint64(123),
	}, nil)
	require.NoError(t, err)

	res, err = h.waitForNotification(ctx)
	require.NoError(t, err)
	d = coap.DetailedResponse[map[string]interface{}]{}
	err = res(&d)
	require.NoError(t, err)
	assert.Equal(t, uint64(123), d.Body["power"].(uint64))
	if ETagSupported {
		require.NotEmpty(t, d.ETag)
		require.NotEqual(t, etag1, d.ETag)
		etag1 = d.ETag
	}

	res, err = h2.waitForNotification(ctx)
	require.NoError(t, err)
	d2 = coap.DetailedResponse[map[string]interface{}]{}
	err = res(&d2)
	require.NoError(t, err)
	assert.Equal(t, uint64(123), d2.Body["power"].(uint64))
	if ETagSupported {
		require.NotEmpty(t, d2.ETag)
		require.NotEqual(t, etag2, d2.ETag)
		require.Equal(t, d.ETag, d2.ETag)
		etag2 = d2.ETag
	}

	err = c.UpdateResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), map[string]interface{}{
		"power": uint64(0),
	}, nil)
	assert.NoError(t, err)

	res, err = h.waitForNotification(ctx)
	require.NoError(t, err)
	d = coap.DetailedResponse[map[string]interface{}]{}
	err = res(&d)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d.Body["power"].(uint64))
	if ETagSupported {
		require.NotEmpty(t, d.ETag)
		require.NotEqual(t, etag1, d.ETag)
	}

	res, err = h2.waitForNotification(ctx)
	require.NoError(t, err)
	d2 = coap.DetailedResponse[map[string]interface{}]{}
	err = res(&d2)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d2.Body["power"].(uint64))
	if ETagSupported {
		require.NotEmpty(t, d2.ETag)
		require.NotEqual(t, etag2, d2.ETag)
		require.Equal(t, d.ETag, d2.ETag)
	}
}

func TestObservingResource(t *testing.T) {
	testDevice(t, test.DevsimName, runObservingResourceTest)
}

func TestObservingDiscoveryResource(t *testing.T) {
	testDevice(t, test.DevsimName, func(ctx context.Context, t *testing.T, c *client.Client, deviceID string) {
		h := makeObservationHandler()
		id, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h)
		require.NoError(t, err)
		d := coap.DetailedResponse[schema.ResourceLinks]{}
		res, err := h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d)
		require.NoError(t, err)
		if ETagSupported {
			require.NotEmpty(t, d.ETag)
		}
		assert.NotEmpty(t, d.Body)
		numResources := len(d.Body)
		d.Body.Sort()
		checkForNonObservationCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		err = h.waitForClose(checkForNonObservationCtx)
		if err == nil {
			// oic/res doesn't support observation
			return
		}
		defer func(observationID string) {
			_, errC := c.StopObservingResource(ctx, observationID)
			require.NoError(t, errC)
		}(id)
		require.Equal(t, context.DeadlineExceeded, err)
		createSwitch(ctx, t, c, deviceID)
		d1 := coap.DetailedResponse[schema.ResourceLinks]{}
		res, err = h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d1)
		require.NoError(t, err)
		if ETagSupported {
			require.NotEmpty(t, d1.ETag)
			require.NotEqual(t, d.ETag, d1.ETag)
		}
		d1.Body.Sort()
		require.Equal(t, numResources+1, len(d1.Body))
		var j int
		for i := range d1.Body {
			if j < len(d.Body) && d.Body[j].Href == d1.Body[i].Href {
				require.Equal(t, d.Body[j], d1.Body[i])
				j++
			} else {
				require.Equal(t, test.TestResourceSwitchesInstanceHref("1"), d1.Body[i].Href)
			}
		}
		err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref("1"), nil)
		require.NoError(t, err)
		d2 := coap.DetailedResponse[schema.ResourceLinks]{}
		res, err = h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d2)
		require.NoError(t, err)
		if ETagSupported {
			require.NotEmpty(t, d2.ETag)
			require.NotEqual(t, d.ETag, d2.ETag)
			require.NotEqual(t, d1.ETag, d2.ETag)
		}
		d2.Body.Sort()
		require.Equal(t, d.Body, d2.Body)
	})
}

func TestObservingDiscoveryResourceWithBaselineInterface(t *testing.T) {
	testDevice(t, test.DevsimName, func(ctx context.Context, t *testing.T, c *client.Client, deviceID string) {
		h := makeObservationHandler()
		id, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_BASELINE))
		require.NoError(t, err)
		d := coap.DetailedResponse[resources.BaselineResourceDiscovery]{}
		res, err := h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d)
		require.NoError(t, err)
		if ETagSupported {
			require.NotEmpty(t, d.ETag)
		}
		require.NotEmpty(t, d.Body)
		numResources := len(d.Body[0].Links)
		d.Body[0].Links.Sort()
		checkForNonObservationCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		err = h.waitForClose(checkForNonObservationCtx)
		if err == nil {
			// oic/res doesn't support observation
			return
		}
		defer func(observationID string) {
			_, errC := c.StopObservingResource(ctx, observationID)
			require.NoError(t, errC)
		}(id)
		require.Equal(t, context.DeadlineExceeded, err)
		createSwitch(ctx, t, c, deviceID)
		d1 := coap.DetailedResponse[resources.BaselineResourceDiscovery]{}
		res, err = h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d1)
		require.NoError(t, err)
		if ETagSupported {
			require.NotEmpty(t, d1.ETag)
			require.NotEqual(t, d.ETag, d1.ETag)
		}
		require.NotEmpty(t, d1.Body)
		d1.Body[0].Links.Sort()
		require.Equal(t, numResources+1, len(d1.Body[0].Links))
		var j int
		for i := range d1.Body {
			if j < len(d.Body) && d.Body[0].Links[j].Href == d1.Body[0].Links[i].Href {
				require.Equal(t, d.Body[0].Links[j], d1.Body[0].Links[i])
				j++
			} else {
				require.Equal(t, test.TestResourceSwitchesInstanceHref("1"), d1.Body[0].Links[i].Href)
			}
		}
		err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref("1"), nil)
		require.NoError(t, err)
		d2 := coap.DetailedResponse[resources.BaselineResourceDiscovery]{}
		res, err = h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d2)
		require.NoError(t, err)
		if ETagSupported {
			require.NotEmpty(t, d1.ETag)
			require.NotEqual(t, d.ETag, d2.ETag)
			require.NotEqual(t, d1.ETag, d2.ETag)
		}
		require.NotEmpty(t, d2.Body)
		d2.Body[0].Links.Sort()
		require.Equal(t, d.Body, d2.Body)
	})
}

func TestObservingNonObservableResource(t *testing.T) {
	testDevice(t, test.DevsimName, func(ctx context.Context, t *testing.T, c *client.Client, deviceID string) {
		h := makeObservationHandler()
		_, err := c.ObserveResource(ctx, deviceID, maintenance.ResourceURI, h)
		require.NoError(t, err)
		d := coap.DetailedResponse[maintenance.Maintenance]{}
		// resource is not observable so action (close/event) depends on goroutine scheduler which is not deterministic
		select {
		case e := <-h.res:
			err = e(&d)
			require.NoError(t, err)
			if ETagSupported {
				require.NotEmpty(t, d.ETag)
			}
			err = h.waitForClose(ctx)
			require.NoError(t, err)
		case <-h.close:
			// if close comes first, then event is not received
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		}
	})
}

func TestObservingDiscoveryResourceWithBatchInterface(t *testing.T) {
	testDevice(t, test.DevsimName, func(ctx context.Context, t *testing.T, c *client.Client, deviceID string) {
		h := makeObservationHandler()
		var v interface{}
		err := c.GetResource(ctx, deviceID, resources.ResourceURI, &v, client.WithInterface(interfaces.OC_IF_LL))
		require.NoError(t, err)

		id, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B))
		require.NoError(t, err)
		var d resources.BatchResourceDiscovery
		res, err := h.waitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d)
		require.NoError(t, err)
		assert.NotEmpty(t, d)
		d.Sort()
		require.Len(t, d, 9)
		for i := range d {
			require.Equal(t, deviceID, d[i].DeviceID())
			switch d[i].Href() {
			case device.ResourceURI:
			case platform.ResourceURI:
			case test.TestResourceLightInstanceHref("1"):
			case cloud.ResourceURI:
			case maintenance.ResourceURI:
			case introspection.ResourceURI:
			case configuration.ResourceURI:
			case test.TestResourceSwitchesHref:
			case plgdtime.ResourceURI:
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
			_, errC := c.StopObservingResource(ctx, observationID)
			require.NoError(t, errC)
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
		require.GreaterOrEqual(t, len(d2), 1)
		for i := range d2 {
			require.Equal(t, deviceID, d2[i].DeviceID())
			switch d2[i].Href() {
			case test.TestResourceSwitchesHref:
			case test.TestResourceSwitchesInstanceHref("1"):
			default:
				require.NoError(t, fmt.Errorf("unknown resource href: %v", d2[i].Href()))
			}
		}
	})
}
