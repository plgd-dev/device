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

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/test"
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

	e, err := h.waitForNotification(ctx)
	require.NoError(t, err)

	test.CheckResourceLinks(t, test.DefaultDevsimResourceLinks(), e)

	err = c.CreateResource(ctx, deviceID, test.TestResourceSwitchesHref, test.MakeSwitchResourceDefaultData(), nil)
	require.NoError(t, err)

	e, err = h.waitForNotification(ctx)
	require.NoError(t, err)
	test.CheckResourceLinks(t, append(test.DefaultDevsimResourceLinks(), test.DefaultSwitchResourceLink("1")), e)

	err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref("1"), nil)
	require.NoError(t, err)

	e, err = h.waitForNotification(ctx)
	require.NoError(t, err)
	test.CheckResourceLinks(t, test.DefaultDevsimResourceLinks(), e)

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
		res:   make(chan schema.ResourceLinks, 100),
		close: make(chan struct{}),
	}
}

type mockDeviceResourcesObservationHandler struct {
	res   chan schema.ResourceLinks
	close chan struct{}
}

func (h *mockDeviceResourcesObservationHandler) Handle(ctx context.Context, body schema.ResourceLinks) {
	h.res <- body
}

func (h *mockDeviceResourcesObservationHandler) Error(err error) {
	fmt.Println(err)
}

func (h *mockDeviceResourcesObservationHandler) OnClose() {
	close(h.close)
}

func (h *mockDeviceResourcesObservationHandler) waitForNotification(ctx context.Context) (schema.ResourceLinks, error) {
	select {
	case e := <-h.res:
		return e, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.close:
		return nil, fmt.Errorf("unexpected close")
	}
}
