/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package bridge_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	bridgeTest "github.com/plgd-dev/device/v2/bridge/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/stretchr/testify/require"
)

func TestGetDevices(t *testing.T) {
	s := bridgeTest.NewBridgeService(t)
	t.Cleanup(func() {
		_ = s.Shutdown()
	})
	deviceID1 := uuid.New().String()
	d1 := bridgeTest.NewBridgedDevice(t, s, deviceID1, false, false)
	deviceID2 := uuid.New().String()
	d2 := bridgeTest.NewBridgedDevice(t, s, deviceID2, false, false)
	deviceID3 := uuid.New().String()
	d3 := bridgeTest.NewBridgedDevice(t, s, deviceID3, false, false)
	defer func() {
		s.DeleteAndCloseDevice(d3.GetID())
		s.DeleteAndCloseDevice(d2.GetID())
		s.DeleteAndCloseDevice(d1.GetID())
	}()

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

	c1, err := testClient.NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c1.Close(context.Background())
		require.NoError(t, errC)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	devices, err := c.GetDevicesDetails(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, devices[deviceID1])
	require.NotEmpty(t, devices[deviceID2])
	require.NotEmpty(t, devices[deviceID3])

	eps := devices[deviceID1].Endpoints
	require.NotEmpty(t, eps)
	addr, err := eps[0].GetAddr()
	require.NoError(t, err)

	ctx1, cancel1 := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel1()

	ipDevs, err := c1.GetDevicesByIP(ctx1, addr.String())
	require.NoError(t, err)
	require.NotEmpty(t, ipDevs)
	devs := make(map[string]bool)
	for _, d := range ipDevs {
		devs[d.Device.DeviceID()] = true
		require.NotEmpty(t, d.Links)
	}
	require.True(t, devs[deviceID1])
	require.True(t, devs[deviceID2])
	require.True(t, devs[deviceID3])
}
