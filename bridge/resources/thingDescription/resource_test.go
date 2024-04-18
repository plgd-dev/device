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

package thingDescription_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device/thingDescription"
	thingDescriptionResource "github.com/plgd-dev/device/v2/bridge/resources/thingDescription"
	bridgeTest "github.com/plgd-dev/device/v2/bridge/test"
	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/stretchr/testify/require"
	wotTD "github.com/web-of-things-open-source/thingdescription-go/thingDescription"
)

func getThingDescription(t *testing.T, data interface{}) wotTD.ThingDescription {
	tdMap, ok := data.(map[interface{}]interface{})
	require.True(t, ok)
	jsonData, err := json.Encode(tdMap)
	require.NoError(t, err)
	td := wotTD.ThingDescription{}
	err = json.Decode(jsonData, &td)
	require.NoError(t, err)
	return td
}

func TestGetThingDescription(t *testing.T) {
	s := bridgeTest.NewBridgeService(t)
	t.Cleanup(func() {
		_ = s.Shutdown()
	})
	deviceID1 := uuid.New().String()
	d1 := bridgeTest.NewBridgedDevice(t, s, deviceID1, true, true, true)
	defer func() {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	devices, err := c.GetDevicesDetails(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, devices[deviceID1])
	eps := devices[deviceID1].Endpoints
	require.NotEmpty(t, eps)

	type args struct {
		deviceID string
		href     string
		opts     []client.GetOption
	}
	tests := []struct {
		name    string
		args    args
		want    wotTD.ThingDescription
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceID: d1.GetID().String(),
				href:     thingDescriptionResource.ResourceURI,
			},
			want: func() wotTD.ThingDescription {
				td, err := bridgeTest.ThingDescription(true, true)
				require.NoError(t, err)
				return thingDescription.PatchThingDescription(td, d1, eps[0].URI, func(resourceHref string, resource thingDescription.Resource) (wotTD.PropertyElement, bool) {
					return bridgeTest.GetPropertyElement(td, d1, eps[0].URI, resourceHref, resource)
				})
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCtx, runCancel := context.WithTimeout(context.Background(), time.Second*8)
			defer runCancel()
			got := coap.DetailedResponse[interface{}]{}
			err := c.GetResource(runCtx, tt.args.deviceID, tt.args.href, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got.ETag = nil
			require.Equal(t, tt.want, getThingDescription(t, got.Body))
		})
	}
}
