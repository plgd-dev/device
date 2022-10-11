package client_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/resources"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/require"
)

func TestObserveDeviceResources(t *testing.T) {
	testDevice(t, test.DevsimName, runObserveDeviceResourcesTest)
}

func isDeviceResourcesObservable(ctx context.Context, t *testing.T, c *client.Client, deviceID string) bool {
	_, links, err := c.GetDevice(ctx, deviceID)
	require.NoError(t, err)
	res := links.GetResourceLinks(resources.ResourceType)
	require.NotEmpty(t, res)
	return res[0].Policy.BitMask.Has(schema.Observable)
}

func runObserveDeviceResourcesTest(t *testing.T, ctx context.Context, c *client.Client, deviceID string) {
	h := makeMockDeviceResourcesObservationHandler()
	if !isDeviceResourcesObservable(ctx, t, c, deviceID) {
		t.Skip("resource is not observable")
		return
	}

	ID, err := c.ObserveDeviceResources(ctx, deviceID, h)
	require.NoError(t, err)

	expLinks := make(map[string]bool)
	for _, l := range test.TestDevsimResources {
		expLinks[l.Href] = true
	}
	for _, l := range test.TestDevsimSecResources {
		expLinks[l.Href] = true
	}
	for _, l := range test.TestDevsimPrivateResources {
		expLinks[l.Href] = true
	}
	for len(expLinks) > 0 {
		e, err := h.waitForNotification(ctx)
		require.NoError(t, err)
		require.Equal(t, client.DeviceResourcesObservationEvent_ADDED, e.Event)
		if _, ok := expLinks[e.Link.Href]; ok {
			delete(expLinks, e.Link.Href)
		} else {
			require.FailNowf(t, "unexpected link", e.Link.Href)
		}
	}

	err = c.CreateResource(ctx, deviceID, test.TestResourceSwitchesHref, test.MakeSwitchResourceDefaultData(), nil)
	require.NoError(t, err)

	e, err := h.waitForNotification(ctx)
	require.NoError(t, err)
	require.Equal(t, client.DeviceResourcesObservationEvent_ADDED, e.Event)
	require.Equal(t, test.TestResourceSwitchesInstanceHref("1"), e.Link.Href)

	err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref("1"), nil)
	require.NoError(t, err)

	e, err = h.waitForNotification(ctx)
	require.NoError(t, err)
	require.Equal(t, client.DeviceResourcesObservationEvent_REMOVED, e.Event)
	require.Equal(t, test.TestResourceSwitchesInstanceHref("1"), e.Link.Href)

	ok, err := c.StopObservingDeviceResources(ctx, ID)
	require.NoError(t, err)
	require.True(t, ok)
	select {
	case <-h.res:
		require.NoError(t, fmt.Errorf("unexpected event"))
	default:
	}
}

func makeMockDeviceResourcesObservationHandler() *mockDeviceResourcesObservationHandler {
	return &mockDeviceResourcesObservationHandler{
		res:   make(chan client.DeviceResourcesObservationEvent, 100),
		close: make(chan struct{}),
	}
}

type mockDeviceResourcesObservationHandler struct {
	res   chan client.DeviceResourcesObservationEvent
	close chan struct{}
}

func (h *mockDeviceResourcesObservationHandler) Handle(ctx context.Context, body client.DeviceResourcesObservationEvent) error {
	h.res <- body
	return nil
}

func (h *mockDeviceResourcesObservationHandler) Error(err error) {
	fmt.Println(err)
}

func (h *mockDeviceResourcesObservationHandler) OnClose() {
	close(h.close)
}

func (h *mockDeviceResourcesObservationHandler) waitForNotification(ctx context.Context) (client.DeviceResourcesObservationEvent, error) {
	select {
	case e := <-h.res:
		return e, nil
	case <-ctx.Done():
		return client.DeviceResourcesObservationEvent{}, ctx.Err()
	case <-h.close:
		return client.DeviceResourcesObservationEvent{}, fmt.Errorf("unexpected close")
	}
}
