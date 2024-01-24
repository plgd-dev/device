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

	"github.com/hashicorp/go-multierror"
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
	"github.com/plgd-dev/device/v2/schema/softwareupdate"
	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDevice(t *testing.T, name string, runTest func(ctx context.Context, t *testing.T, c *client.Client, deviceID string)) {
	deviceID := test.MustFindDeviceByName(name)
	c, err := testClient.NewTestSecureClient()
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
	h := testClient.MakeMockResourceObservationHandler()
	id, err := c.ObserveResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), h)
	require.NoError(t, err)
	defer func(observationID string) {
		_, errC := c.StopObservingResource(ctx, observationID)
		require.NoError(t, errC)
	}(id)

	res, err := h.WaitForNotification(ctx)
	require.NoError(t, err)
	d := coap.DetailedResponse[map[string]interface{}]{}
	err = res(&d)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d.Body["power"].(uint64))
	etag1 := d.ETag
	if ETagSupported {
		assert.NotEmpty(t, d.ETag)
	}

	h2 := testClient.MakeMockResourceObservationHandler()
	id, err = c.ObserveResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), h2)
	require.NoError(t, err)
	defer func(observationID string) {
		_, errC := c.StopObservingResource(ctx, observationID)
		require.NoError(t, errC)
	}(id)

	res, err = h2.WaitForNotification(ctx)
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

	res, err = h.WaitForNotification(ctx)
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

	res, err = h2.WaitForNotification(ctx)
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

	res, err = h.WaitForNotification(ctx)
	require.NoError(t, err)
	d = coap.DetailedResponse[map[string]interface{}]{}
	err = res(&d)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), d.Body["power"].(uint64))
	if ETagSupported {
		require.NotEmpty(t, d.ETag)
		require.NotEqual(t, etag1, d.ETag)
	}

	res, err = h2.WaitForNotification(ctx)
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
		h := testClient.MakeMockResourceObservationHandler()
		id, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h)
		require.NoError(t, err)
		d := coap.DetailedResponse[schema.ResourceLinks]{}
		res, err := h.WaitForNotification(ctx)
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
		err = h.WaitForClose(checkForNonObservationCtx)
		if err == nil {
			// oic/res doesn't support observation
			return
		}
		defer func(observationID string) {
			_, errC := c.StopObservingResource(ctx, observationID)
			require.NoError(t, errC)
		}(id)
		require.Equal(t, context.DeadlineExceeded, err)
		const switchID = "1"
		createSwitches(ctx, t, c, deviceID, 1)
		d1 := coap.DetailedResponse[schema.ResourceLinks]{}
		res, err = h.WaitForNotification(ctx)
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
				require.Equal(t, test.TestResourceSwitchesInstanceHref(switchID), d1.Body[i].Href)
			}
		}
		err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref(switchID), nil)
		require.NoError(t, err)
		d2 := coap.DetailedResponse[schema.ResourceLinks]{}
		res, err = h.WaitForNotification(ctx)
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
		h := testClient.MakeMockResourceObservationHandler()
		id, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_BASELINE))
		require.NoError(t, err)
		d := coap.DetailedResponse[resources.BaselineResourceDiscovery]{}
		res, err := h.WaitForNotification(ctx)
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
		err = h.WaitForClose(checkForNonObservationCtx)
		if err == nil {
			// oic/res doesn't support observation
			return
		}
		defer func(observationID string) {
			_, errC := c.StopObservingResource(ctx, observationID)
			require.NoError(t, errC)
		}(id)
		require.Equal(t, context.DeadlineExceeded, err)
		const switchID = "1"
		createSwitches(ctx, t, c, deviceID, 1)
		d1 := coap.DetailedResponse[resources.BaselineResourceDiscovery]{}
		res, err = h.WaitForNotification(ctx)
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
				require.Equal(t, test.TestResourceSwitchesInstanceHref(switchID), d1.Body[0].Links[i].Href)
			}
		}
		err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref(switchID), nil)
		require.NoError(t, err)
		d2 := coap.DetailedResponse[resources.BaselineResourceDiscovery]{}
		res, err = h.WaitForNotification(ctx)
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
		h := testClient.MakeMockResourceObservationHandler()
		_, err := c.ObserveResource(ctx, deviceID, maintenance.ResourceURI, h)
		require.NoError(t, err)
		d := coap.DetailedResponse[maintenance.Maintenance]{}
		// resource is not observable so action (close/event) depends on goroutine scheduler which is not deterministic
		select {
		case e := <-h.Res:
			err = e(&d)
			require.NoError(t, err)
			if ETagSupported {
				require.NotEmpty(t, d.ETag)
			}
			err = h.WaitForClose(ctx)
			require.NoError(t, err)
		case <-h.Close:
			// if close comes first, then event is not received
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		}
	})
}

type batchResource struct {
	href    string
	deleted bool
}

func verifyBatchDiscoveryResponse(t *testing.T, deviceID string, resp coap.DetailedResponse[resources.BatchResourceDiscovery], code codes.Code, expected ...batchResource) {
	require.Equal(t, code, resp.Code)
	if code == codes.Valid {
		require.Empty(t, resp.Body)
		return
	}

	hrefs_len := len(expected)
	expectedResources := make(map[string]bool, hrefs_len)
	for _, batch := range expected {
		expectedResources[batch.href] = batch.deleted
	}

	resp.Body.Sort()
	require.Len(t, resp.Body, hrefs_len)

	bodyHrefs := make(map[string]struct{}, hrefs_len)
	for i := range resp.Body {
		require.Equal(t, deviceID, resp.Body[i].DeviceID())
		deleted, ok := expectedResources[resp.Body[i].Href()]
		if !ok {
			require.NoError(t, fmt.Errorf("unknown resource href: %v", resp.Body[i].Href()))
		}
		require.NotEmpty(t, resp.Body[i].Content)
		if ETagSupported {
			if deleted {
				require.Empty(t, resp.Body[i].ETag)
			} else {
				require.NotEmpty(t, resp.Body[i].ETag)
			}
		}
		bodyHrefs[resp.Body[i].Href()] = struct{}{}
	}

	for _, batch := range expected {
		if _, ok := bodyHrefs[batch.href]; !ok {
			require.NoError(t, fmt.Errorf("missing resource href: %v", batch.href))
		}
	}
}

var expectedFirstBatchResources = []batchResource{
	{device.ResourceURI, false},
	{platform.ResourceURI, false},
	{test.TestResourceLightInstanceHref("1"), false},
	{cloud.ResourceURI, false},
	{maintenance.ResourceURI, false},
	{introspection.ResourceURI, false},
	{configuration.ResourceURI, false},
	{test.TestResourceSwitchesHref, false},
	{plgdtime.ResourceURI, false},
	{softwareupdate.ResourceURI, false},
}

func TestObservingDiscoveryResourceWithBatchInterface(t *testing.T) {
	testDevice(t, test.DevsimName, func(ctx context.Context, t *testing.T, c *client.Client, deviceID string) {
		h := testClient.MakeMockResourceObservationHandler()
		id, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B))
		require.NoError(t, err)
		var d coap.DetailedResponse[resources.BatchResourceDiscovery]
		res, err := h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d)
		require.NoError(t, err)
		assert.NotEmpty(t, d.Body)
		expected := expectedFirstBatchResources
		verifyBatchDiscoveryResponse(t, deviceID, d, codes.Content, expected...)
		if ETagSupported {
			require.NotEmpty(t, d.ETag)
		}
		checkForNonObservationCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		err = h.WaitForClose(checkForNonObservationCtx)
		if err == nil {
			// oic/res doesn't support observation
			return
		}
		defer func(observationID string) {
			_, errC := c.StopObservingResource(ctx, observationID)
			require.NoError(t, errC)
		}(id)
		require.Equal(t, context.DeadlineExceeded, err)
		const switchID = "1"
		createSwitches(ctx, t, c, deviceID, 1)
		var d1 coap.DetailedResponse[resources.BatchResourceDiscovery]
		res, err = h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d1)
		require.NoError(t, err)
		changed := []batchResource{
			{test.TestResourceSwitchesInstanceHref(switchID), false},
			{test.TestResourceSwitchesHref, false},
		}
		verifyBatchDiscoveryResponse(t, deviceID, d1, codes.Content, changed...)
		if ETagSupported {
			require.NotEmpty(t, d1.ETag)
			require.NotEqual(t, d.ETag, d1.ETag)
		}
		err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref(switchID), nil)
		require.NoError(t, err)
		var d2 coap.DetailedResponse[resources.BatchResourceDiscovery]
		res, err = h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d2)
		require.NoError(t, err)
		d2.Body.Sort()
		require.GreaterOrEqual(t, len(d2.Body), 1)
		if ETagSupported {
			require.NotEmpty(t, d2.ETag)
			require.NotEqual(t, d.ETag, d2.ETag)
			require.NotEqual(t, d1.ETag, d2.ETag)
		}
		for i := range d2.Body {
			require.Equal(t, deviceID, d2.Body[i].DeviceID())
			switch d2.Body[i].Href() {
			case test.TestResourceSwitchesHref:
				if ETagSupported {
					require.NotEmpty(t, d2.Body[i].ETag)
				}
			case test.TestResourceSwitchesInstanceHref("1"):
				if ETagSupported {
					require.Empty(t, d2.Body[i].ETag)
				}
			default:
				require.NoError(t, fmt.Errorf("unknown resource href: %v", d2.Body[i].Href()))
			}
		}
	})
}

func TestObserveDiscoveryResourceWithIncrementalChangesOnUpdate(t *testing.T) {
	if !ETagIncrementalChangesSupported {
		t.Skip("incremental changes not supported")
	}

	testDevice(t, test.DevsimName, func(ctx context.Context, t *testing.T, c *client.Client, deviceID string) {
		h := testClient.MakeMockResourceObservationHandler()
		obsID, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B))
		require.NoError(t, err)
		d0 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		res, err := h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d0)
		require.NoError(t, err)
		assert.NotEmpty(t, d0.Body)
		expected := expectedFirstBatchResources
		require.NotEmpty(t, d0.ETag)
		verifyBatchDiscoveryResponse(t, deviceID, d0, codes.Content, expected...)
		etags := make([][]byte, 0, len(d0.Body)-1)
		for _, resource := range d0.Body {
			if resource.Href() == test.TestResourceLightInstanceHref("1") {
				continue
			}
			etags = append(etags, resource.ETag)
		}

		checkForNonObservationCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		err = h.WaitForClose(checkForNonObservationCtx)
		if err == nil {
			t.Skip("oic/res doesn't support observation")
			return
		}

		err = c.UpdateResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), map[string]interface{}{
			"power": uint64(123),
		}, nil)
		require.NoError(t, err)
		defer func() {
			// restore to original value
			errU := c.UpdateResource(ctx, deviceID, test.TestResourceLightInstanceHref("1"), map[string]interface{}{
				"power": uint64(0),
			}, nil)
			require.NoError(t, errU)
		}()

		res, err = h.WaitForNotification(ctx)
		require.NoError(t, err)
		d1 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		err = res(&d1)
		require.NoError(t, err)
		require.NotEqual(t, d0.ETag, d1.ETag)
		changed := []batchResource{
			{test.TestResourceLightInstanceHref("1"), false},
		}
		verifyBatchDiscoveryResponse(t, deviceID, d1, codes.Content, changed...)
		require.Equal(t, d1.ETag, d1.Body[0].ETag)

		_, err = c.StopObservingResource(ctx, obsID)
		require.NoError(t, err)

		queries := coap.EncodeETagsForIncrementalChanges([][]byte{d1.ETag})
		require.Equal(t, 1, len(queries))
		obsID, err = c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B), client.WithQuery(queries[0]))
		require.NoError(t, err)
		d2 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		res, err = h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d2)
		require.NoError(t, err)
		require.Equal(t, d1.ETag, d2.ETag)
		verifyBatchDiscoveryResponse(t, deviceID, d2, codes.Valid)

		_, errC := c.StopObservingResource(ctx, obsID)
		require.NoError(t, errC)

		// every resource except the updated switch should match original etags, thus only the updated switch should be in the payload
		queries = coap.EncodeETagsForIncrementalChanges(etags)
		require.Equal(t, 1, len(queries))
		obsID, err = c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B), client.WithQuery(queries[0]))
		require.NoError(t, err)
		defer func(observationID string) {
			_, errC := c.StopObservingResource(ctx, observationID)
			require.NoError(t, errC)
		}(obsID)
		d3 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		res, err = h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d3)
		require.NoError(t, err)
		require.Equal(t, d2.ETag, d3.ETag)
		changed = []batchResource{
			{test.TestResourceLightInstanceHref("1"), false},
		}
		verifyBatchDiscoveryResponse(t, deviceID, d3, codes.Content, changed...)
	})
}

func TestObserveDiscoveryResourceWithIncrementalChangesOnCreate(t *testing.T) {
	if !ETagIncrementalChangesSupported {
		t.Skip("incremental changes not supported")
	}

	testDevice(t, test.DevsimName, func(ctx context.Context, t *testing.T, c *client.Client, deviceID string) {
		h := testClient.MakeMockResourceObservationHandler()
		obsID, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B))
		require.NoError(t, err)
		d0 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		res, err := h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d0)
		require.NoError(t, err)
		assert.NotEmpty(t, d0.Body)
		expected := expectedFirstBatchResources
		require.NotEmpty(t, d0.ETag)
		verifyBatchDiscoveryResponse(t, deviceID, d0, codes.Content, expected...)
		etags := make([][]byte, 0, len(d0.Body)-1)
		for _, resource := range d0.Body {
			if resource.Href() == test.TestResourceSwitchesHref {
				continue
			}
			etags = append(etags, resource.ETag)
		}

		checkForNonObservationCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		err = h.WaitForClose(checkForNonObservationCtx)
		if err == nil {
			t.Skip("oic/res doesn't support observation")
			return
		}

		const switchID = "1"
		createSwitches(ctx, t, c, deviceID, 1)
		defer func() {
			errD := c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref(switchID), nil)
			require.NoError(t, errD)
		}()

		res, err = h.WaitForNotification(ctx)
		require.NoError(t, err)
		d1 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		err = res(&d1)
		require.NoError(t, err)
		require.NotEqual(t, d0.ETag, d1.ETag)
		changed := []batchResource{
			{test.TestResourceSwitchesInstanceHref(switchID), false},
			{test.TestResourceSwitchesHref, false},
		}
		verifyBatchDiscoveryResponse(t, deviceID, d1, codes.Content, changed...)

		_, err = c.StopObservingResource(ctx, obsID)
		require.NoError(t, err)

		queries := coap.EncodeETagsForIncrementalChanges([][]byte{d1.ETag})
		require.Equal(t, 1, len(queries))
		obsID, err = c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B), client.WithQuery(queries[0]))
		require.NoError(t, err)
		d2 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		res, err = h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d2)
		require.NoError(t, err)
		require.Equal(t, d1.ETag, d2.ETag)
		verifyBatchDiscoveryResponse(t, deviceID, d2, codes.Valid)

		_, errC := c.StopObservingResource(ctx, obsID)
		require.NoError(t, errC)

		// every resource except the updated /switches resources should match original etags
		queries = coap.EncodeETagsForIncrementalChanges(etags)
		require.Equal(t, 1, len(queries))
		obsID, err = c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B), client.WithQuery(queries[0]))
		require.NoError(t, err)
		defer func(observationID string) {
			_, errC := c.StopObservingResource(ctx, observationID)
			require.NoError(t, errC)
		}(obsID)
		d3 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		res, err = h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d3)
		require.NoError(t, err)
		require.Equal(t, d2.ETag, d3.ETag)
		changed = []batchResource{
			{test.TestResourceSwitchesInstanceHref(switchID), false},
			{test.TestResourceSwitchesHref, false},
		}
		verifyBatchDiscoveryResponse(t, deviceID, d3, codes.Content, changed...)
	})
}

func TestObserveDiscoveryResourceWithIncrementalChangesOnDelete(t *testing.T) {
	if !ETagIncrementalChangesSupported {
		t.Skip("incremental changes not supported")
	}
	testDevice(t, test.DevsimName, func(ctx context.Context, t *testing.T, c *client.Client, deviceID string) {
		const switchID = "1"
		createSwitches(ctx, t, c, deviceID, 1)
		toDelete := []string{test.TestResourceSwitchesInstanceHref(switchID)}
		defer func() {
			var errs *multierror.Error
			for _, href := range toDelete {
				if err := c.DeleteResource(ctx, deviceID, href, nil); err != nil {
					errs = multierror.Append(errs, err)
				}
			}
			require.NoError(t, errs.ErrorOrNil())
		}()

		h := testClient.MakeMockResourceObservationHandler()
		obsID, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B))
		require.NoError(t, err)
		d0 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		res, err := h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d0)
		require.NoError(t, err)
		assert.NotEmpty(t, d0.Body)
		expected := expectedFirstBatchResources
		expected = append(expected, batchResource{test.TestResourceSwitchesInstanceHref(switchID), false})
		require.NotEmpty(t, d0.ETag)
		verifyBatchDiscoveryResponse(t, deviceID, d0, codes.Content, expected...)
		etags := make([][]byte, 0, len(d0.Body)-1)
		for _, resource := range d0.Body {
			if resource.Href() == test.TestResourceSwitchesHref {
				continue
			}
			etags = append(etags, resource.ETag)
		}

		checkForNonObservationCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		err = h.WaitForClose(checkForNonObservationCtx)
		if err == nil {
			t.Skip("oic/res doesn't support observation")
			return
		}

		errD := c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref(switchID), nil)
		require.NoError(t, errD)
		toDelete = nil

		res, err = h.WaitForNotification(ctx)
		require.NoError(t, err)
		d1 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		err = res(&d1)
		require.NoError(t, err)
		require.NotEqual(t, d0.ETag, d1.ETag)
		changed := []batchResource{
			{test.TestResourceSwitchesHref, false},
			{test.TestResourceSwitchesInstanceHref(switchID), true},
		}
		verifyBatchDiscoveryResponse(t, deviceID, d1, codes.Content, changed...)

		_, err = c.StopObservingResource(ctx, obsID)
		require.NoError(t, err)

		queries := coap.EncodeETagsForIncrementalChanges([][]byte{d1.ETag})
		require.Equal(t, 1, len(queries))
		obsID, err = c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B), client.WithQuery(queries[0]))
		require.NoError(t, err)
		d2 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		res, err = h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d2)
		require.NoError(t, err)
		require.Equal(t, d1.ETag, d2.ETag)
		verifyBatchDiscoveryResponse(t, deviceID, d2, codes.Valid)

		_, errC := c.StopObservingResource(ctx, obsID)
		require.NoError(t, errC)

		// every resource except the updated /switches resources should match original etags
		queries = coap.EncodeETagsForIncrementalChanges(etags)
		require.Equal(t, 1, len(queries))
		obsID, err = c.ObserveResource(ctx, deviceID, resources.ResourceURI, h, client.WithInterface(interfaces.OC_IF_B), client.WithQuery(queries[0]))
		require.NoError(t, err)
		defer func(observationID string) {
			_, errC := c.StopObservingResource(ctx, observationID)
			require.NoError(t, errC)
		}(obsID)
		d3 := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		res, err = h.WaitForNotification(ctx)
		require.NoError(t, err)
		err = res(&d3)
		require.NoError(t, err)
		require.Equal(t, d2.ETag, d3.ETag)
		changed = []batchResource{
			{test.TestResourceSwitchesHref, false},
		}
		verifyBatchDiscoveryResponse(t, deviceID, d3, codes.Content, changed...)
	})
}
