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
	"bytes"
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	bridgeTest "github.com/plgd-dev/device/v2/bridge/test"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	codecOcf "github.com/plgd-dev/device/v2/pkg/codec/ocf"
	ocfCloud "github.com/plgd-dev/device/v2/pkg/ocf/cloud"
	cloudSchema "github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	mockCoapGW "github.com/plgd-dev/device/v2/test/coap-gateway"
	mockCoapGWService "github.com/plgd-dev/device/v2/test/coap-gateway/service"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/message/status"
	"github.com/stretchr/testify/require"
)

type resourceData struct {
	Name string `json:"name,omitempty"`
}

type resourceDataSync struct {
	resourceData
	lock sync.Mutex
}

func (r *resourceDataSync) setName(name string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.Name = name
}

func (r *resourceDataSync) copy() resourceData {
	r.lock.Lock()
	defer r.lock.Unlock()
	return resourceData{
		Name: r.Name,
	}
}

func getUnauthorizedError() status.Status {
	msg := pool.NewMessage(context.Background())
	msg.SetCode(codes.Unauthorized)
	return status.Errorf(msg, "unauthorized")
}

func TestManagerDeviceBecomesUnauthorized(t *testing.T) {
	ch := mockCoapGW.NewCoapHandlerWithCounter(3600)
	customHandler := mockCoapGW.NewCustomHandler(ch)
	makeHandler := func(*mockCoapGWService.Service, ...mockCoapGWService.Option) mockCoapGWService.ServiceHandler {
		return customHandler
	}
	coapShutdown := mockCoapGW.New(t, makeHandler, func(mockCoapGWService.ServiceHandler) {
		h := ch
		fmt.Printf("%+v\n", h.CallCounter.Data)
		// d1 -> signup + signin + publish
		// d2 -> should use the stored credentials to skip signup and only do sign in + publish
		require.Equal(t, 1, h.CallCounter.Data[mockCoapGW.SignUpKey])
		require.Equal(t, 1, h.CallCounter.Data[mockCoapGW.SignInKey])
		require.Equal(t, 1, h.CallCounter.Data[mockCoapGW.PublishKey])
		require.Equal(t, 0, h.CallCounter.Data[mockCoapGW.UnpublishKey])
		require.Equal(t, 0, h.CallCounter.Data[mockCoapGW.RefreshTokenKey])
	})
	defer coapShutdown()

	s1 := bridgeTest.NewBridgeService(t)
	t.Cleanup(func() {
		_ = s1.Shutdown()
	})
	deviceID := uuid.New().String()
	tickInterval := time.Second
	d1 := bridgeTest.NewBridgedDevice(t, s1, deviceID, true, false, device.WithCloudOptions(cloud.WithTickInterval(tickInterval)))
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
	err = c.OnboardDevice(ctx, deviceID, "authorizationProvider", "coaps+tcp://"+mockCoapGW.COAP_GW_HOST, "authorizationCode", test.CloudSID())
	require.NoError(t, err)

	// wait for sign in
	require.Equal(t, 1, ch.WaitForSignIn(time.Second*20))

	// wait for publish
	require.Equal(t, 1, ch.WaitForPublish(time.Second*20))

	customHandler.SetSignIn(func(ocfCloud.CoapSignInRequest) (ocfCloud.CoapSignInResponse, error) {
		return ocfCloud.CoapSignInResponse{}, getUnauthorizedError()
	})
	customHandler.SetRefreshToken(func(ocfCloud.CoapRefreshTokenRequest) (ocfCloud.CoapRefreshTokenResponse, error) {
		return ocfCloud.CoapRefreshTokenResponse{}, getUnauthorizedError()
	})

	d1.GetCloudManager().Reconnect()
	for i := 0; i < 5; i++ {
		cfg := d1.GetCloudManager().ExportConfig()
		if cfg.AccessToken == "" {
			return
		}
		time.Sleep(tickInterval * 2)
	}
	require.Fail(t, "cloud manager should be reset, but it is not")
}

func TestProvisioningOnDeviceRestart(t *testing.T) {
	ch := mockCoapGW.NewCoapHandlerWithCounter(-1)
	makeHandler := func(*mockCoapGWService.Service, ...mockCoapGWService.Option) mockCoapGWService.ServiceHandler {
		return ch
	}
	coapShutdown := mockCoapGW.New(t, makeHandler, func(handler mockCoapGWService.ServiceHandler) {
		h := handler.(*mockCoapGW.DefaultHandlerWithCounter)
		fmt.Printf("%+v\n", h.CallCounter.Data)
		// d1 -> signup + signin + publish
		// d2 -> should use the stored credentials to skip signup and only do sign in + publish
		require.Equal(t, 1, h.CallCounter.Data[mockCoapGW.SignUpKey])
		require.Equal(t, 2, h.CallCounter.Data[mockCoapGW.SignInKey])
		require.Equal(t, 3, h.CallCounter.Data[mockCoapGW.PublishKey])
		require.Equal(t, 1, h.CallCounter.Data[mockCoapGW.UnpublishKey])
		require.Equal(t, 0, h.CallCounter.Data[mockCoapGW.RefreshTokenKey])
	})
	defer coapShutdown()

	s1 := bridgeTest.NewBridgeService(t)
	t.Cleanup(func() {
		_ = s1.Shutdown()
	})
	deviceID := uuid.New().String()
	d1 := bridgeTest.NewBridgedDevice(t, s1, deviceID, true, false)
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
	err = c.OnboardDevice(ctx, deviceID, "authorizationProvider", "coaps+tcp://"+mockCoapGW.COAP_GW_HOST, "authorizationCode", test.CloudSID())
	require.NoError(t, err)

	// wait for sign in
	require.Equal(t, 1, ch.WaitForSignIn(time.Second*20))

	// wait for publish
	require.Equal(t, 1, ch.WaitForPublish(time.Second*20))

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
	// wait for publish
	require.Equal(t, 2, ch.WaitForPublish(time.Second*20))

	// check provisioning status
	var cloudCfg cloud.Configuration
	err = c.GetResource(ctx, deviceID, cloudSchema.ResourceURI, &cloudCfg)
	require.NoError(t, err)
	require.Equal(t, cloudSchema.ProvisioningStatus_REGISTERED, cloudCfg.ProvisioningStatus)

	rds := resourceDataSync{
		resourceData: resourceData{
			Name: "test",
		},
	}

	resHandler := func(req *net.Request) (*pool.Message, error) {
		resp := pool.NewMessage(req.Context())
		switch req.Code() {
		case codes.GET:
			resp.SetCode(codes.Content)
		case codes.POST:
			resp.SetCode(codes.Changed)
		default:
			return nil, fmt.Errorf("invalid method %v", req.Code())
		}
		resp.SetContentFormat(message.AppOcfCbor)
		data, err := cbor.Encode(rds.copy())
		if err != nil {
			return nil, err
		}
		resp.SetBody(bytes.NewReader(data))
		return resp, nil
	}

	res := resources.NewResource("/test", resHandler, func(req *net.Request) (*pool.Message, error) {
		codec := codecOcf.VNDOCFCBORCodec{}
		var newData resourceData
		err := codec.Decode(req.Message, &newData)
		if err != nil {
			return nil, err
		}
		rds.setName(newData.Name)
		return resHandler(req)
	}, []string{"oic.d.virtual", "oic.d.test"}, []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_RW})
	res.SetObserveHandler(d2.GetLoop(), func(req *net.Request, handler func(msg *pool.Message, err error)) (cancel func(), err error) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			defer cancel()
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Millisecond * 100):
					resp, err := resHandler(req)
					if err != nil {
						handler(nil, err)
						return
					}
					handler(resp, nil)
				}
			}
		}()
		return cancel, nil
	})

	d2.AddResources(res)
	// wait for publish
	require.Equal(t, 3, ch.WaitForPublish(time.Second*20))

	require.True(t, d2.CloseAndDeleteResource(res.GetHref()))
	// wait for unpublish
	require.Equal(t, 1, ch.WaitForUnpublish(time.Second*20))

	// sign off
	cloudManager := d2.GetCloudManager()
	if cloudManager != nil {
		cloudManager.Unregister()
	}
	require.Equal(t, 1, ch.WaitForSignOff(time.Second*20))
}
