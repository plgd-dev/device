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
	"errors"
	"testing"

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
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

func runObserveDeviceResourcesTest(ctx context.Context, t *testing.T, c *client.Client, deviceID string) {
	h := testClient.MakeMockDeviceResourcesObservationHandler()
	if !isDeviceResourcesObservable(ctx, t, c, deviceID) {
		t.Skip("resource is not observable")
		return
	}

	ID, err := c.ObserveDeviceResources(ctx, deviceID, h)
	require.NoError(t, err)

	e, err := h.WaitForNotification(ctx)
	require.NoError(t, err)

	test.CheckResourceLinks(t, test.DefaultDevsimResourceLinks(), e)

	err = c.CreateResource(ctx, deviceID, test.TestResourceSwitchesHref, test.MakeSwitchResourceDefaultData(), nil)
	require.NoError(t, err)

	e, err = h.WaitForNotification(ctx)
	require.NoError(t, err)
	test.CheckResourceLinks(t, append(test.DefaultDevsimResourceLinks(), test.DefaultSwitchResourceLink("1")), e)

	err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref("1"), nil)
	require.NoError(t, err)

	e, err = h.WaitForNotification(ctx)
	require.NoError(t, err)
	test.CheckResourceLinks(t, test.DefaultDevsimResourceLinks(), e)

	ok, err := c.StopObservingDeviceResources(ctx, ID)
	require.NoError(t, err)
	require.True(t, ok)

	err = h.WaitForClose(ctx)
	require.True(t, err == nil || errors.Is(err, context.DeadlineExceeded))
}
