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

package cloud_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	bridgeTest "github.com/plgd-dev/device/v2/bridge/test"
	cloudSchema "github.com/plgd-dev/device/v2/schema/cloud"
	testClient "github.com/plgd-dev/device/v2/test/client"
	mockCoapGW "github.com/plgd-dev/device/v2/test/coap-gateway"
	mockCoapGWService "github.com/plgd-dev/device/v2/test/coap-gateway/service"
	"github.com/stretchr/testify/require"
)

// device is restarted with an imported configuration with valid cloud credentials
func TestProvisioningOnDeviceRestart(t *testing.T) {
	ch := mockCoapGW.NewCoapHandlerWithCounter(-1)
	makeHandler := func(s *mockCoapGWService.Service, opts ...mockCoapGWService.Option) mockCoapGWService.ServiceHandler {
		return ch
	}
	coapShutdown := mockCoapGW.New(t, makeHandler, func(handler mockCoapGWService.ServiceHandler) {
		h := handler.(*mockCoapGW.DefaultHandlerWithCounter)
		fmt.Printf("%+v\n", h.CallCounter.Data)
		// d1 -> signup + signin + publish
		// d2 -> should use the stored credentials to skip signup and only do sign in + publish
		require.Equal(t, 1, h.CallCounter.Data[mockCoapGW.SignUpKey])
		require.Equal(t, 2, h.CallCounter.Data[mockCoapGW.SignInKey])
		require.Equal(t, 2, h.CallCounter.Data[mockCoapGW.PublishKey])
		require.Equal(t, 0, h.CallCounter.Data[mockCoapGW.RefreshTokenKey])
	})
	defer coapShutdown()

	s1 := bridgeTest.NewBridgeService(t)
	t.Cleanup(func() {
		_ = s1.Shutdown()
	})
	deviceID := uuid.New().String()
	d1 := bridgeTest.NewBridgedDevice(t, s1, true, deviceID)
	s1Shutdown := bridgeTest.RunBridgeService(s1)
	t.Cleanup(func() {
		_ = s1Shutdown()
	})

	c, err := testClient.NewTestSecureClientWithBridgeSupport()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err = c.OnboardDevice(ctx, deviceID, "authorizationProvider", "coaps+tcp://"+mockCoapGW.COAP_GW_HOST, "authorizationCode", "cloudID")
	require.NoError(t, err)

	// wait for sign in
	require.Equal(t, 1, ch.WaitForSignIn(time.Second*20))

	// stop service
	err = s1Shutdown()
	require.NoError(t, err)

	// save the device configuration
	cfg := d1.ExportConfig()

	// recreate device using the saved configuration from a signed in device
	s2 := bridgeTest.NewBridgeService(t)
	t.Cleanup(func() {
		_ = s2.Shutdown()
	})
	d2 := bridgeTest.NewBridgedDeviceWithConfig(t, s2, cfg)
	s2Shutdown := bridgeTest.RunBridgeService(s2)
	defer func() {
		errS := s2Shutdown()
		require.NoError(t, errS)
	}()
	require.Equal(t, 2, ch.WaitForSignIn(time.Second*20))

	// check provisioning status
	var cloudCfg cloud.Configuration
	err = c.GetResource(ctx, deviceID, cloudSchema.ResourceURI, &cloudCfg)
	require.NoError(t, err)
	require.Equal(t, cloudCfg.ProvisioningStatus, cloudSchema.ProvisioningStatus_REGISTERED)

	// sign off
	d2.UnregisterFromCloud()
	require.Equal(t, 1, ch.WaitForSignOff(time.Second*20))
}
