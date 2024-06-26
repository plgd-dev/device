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
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceDiscovery(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := testClient.NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	devices, err := c.GetDevicesDetails(ctx)
	require.NoError(t, err)

	d := devices[deviceID]
	require.NotEmpty(t, d)
	dev, ok := d.Details.(*device.Device)
	require.True(t, ok)
	require.Equal(t, test.DevsimName, dev.Name)

	d = devices[secureDeviceID]
	fmt.Println(d)
	require.NotNil(t, d)
	dev, ok = d.Details.(*device.Device)
	require.True(t, ok)
	require.Equal(t, test.DevsimName, dev.Name)
	require.NotNil(t, d.Ownership)
	require.Equal(t, "00000000-0000-0000-0000-000000000000", d.Ownership.OwnerID)

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	secureDeviceID, err = c.OwnDevice(ctx, secureDeviceID, client.WithOTMs([]client.OTMType{client.OTMType_JustWorks}))
	require.NoError(t, err)
	devices, err = c.GetDevicesDetails(ctx)
	require.NoError(t, err)

	d = devices[secureDeviceID]
	fmt.Println(d)
	require.NotNil(t, d)
	require.NotNil(t, d.Ownership)
	sdkID, err := c.CoreClient().GetSdkOwnerID()
	require.NoError(t, err)
	require.Equal(t, d.Ownership.OwnerID, sdkID)

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = c.DisownDevice(ctx, secureDeviceID)
	require.NoError(t, err)
}

func TestDeviceDiscoveryWithFilter(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := testClient.NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	devices, err := c.GetDevicesDetails(ctx, client.WithResourceTypes(device.ResourceType))
	require.NoError(t, err)
	assert.NotEmpty(t, devices[secureDeviceID], "unreachable test device")

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	devices, err = c.GetDevicesDetails(ctx, client.WithResourceTypes("x.com.device"))
	require.NoError(t, err)
	assert.Empty(t, devices, "test device not filtered out")
}

func TestDevicesWithFoundByIP(t *testing.T) {
	ip4 := test.MustFindDeviceIP(test.DevsimName, test.IP4)
	c, err := testClient.NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	dev, err := c.GetDeviceDetailsByIP(ctx, ip4)
	require.NoError(t, err)
	require.NotNil(t, dev)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	devices, err := c.GetDevicesDetails(ctx, client.WithDiscoveryConfiguration(core.DiscoveryConfiguration{}))
	require.NoError(t, err)
	assert.Equal(t, devices[dev.ID], dev)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	devices, err = c.GetDevicesDetails(ctx, client.WithDiscoveryConfiguration(core.DiscoveryConfiguration{}), client.WithResourceTypes(device.ResourceType))
	require.NoError(t, err)
	assert.Equal(t, devices[dev.ID], dev)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	devices, err = c.GetDevicesDetails(ctx, client.WithDiscoveryConfiguration(core.DiscoveryConfiguration{}), client.WithResourceTypes("x.com.device"))
	require.NoError(t, err)
	assert.Empty(t, devices, "test device not filtered out")

	require.True(t, c.DeleteDevice(ctx, dev.ID))
	devices, err = c.GetDevicesDetails(ctx, client.WithDiscoveryConfiguration(core.DiscoveryConfiguration{}), client.WithResourceTypes(device.ResourceType))
	require.NoError(t, err)
	assert.Empty(t, devices, "empty devices are expected")
}
