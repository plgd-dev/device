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
	"github.com/plgd-dev/device/v2/bridge/resources"
	bridgeTest "github.com/plgd-dev/device/v2/bridge/test"
	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/stretchr/testify/require"
)

func TestGetResource(t *testing.T) {
	s := bridgeTest.NewBridgeService(t)
	t.Cleanup(func() {
		_ = s.Shutdown()
	})
	deviceID1 := uuid.New().String()
	d1 := bridgeTest.NewBridgedDevice(t, s, deviceID1, false, false)
	deviceID2 := uuid.New().String()
	d2 := bridgeTest.NewBridgedDevice(t, s, deviceID2, false, false)
	defer func() {
		s.DeleteAndCloseDevice(d2.GetID())
		s.DeleteAndCloseDevice(d1.GetID())
	}()

	failRes := resources.NewResource("/fail",
		nil,
		nil,
		[]string{"oic.d.virtual", "oic.d.test"},
		[]string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_R},
	)
	d1.AddResources(failRes)

	cleanup := bridgeTest.RunBridgeService(s)
	defer func() {
		errC := cleanup()
		require.NoError(t, errC)
	}()

	type args struct {
		deviceID string
		href     string
		opts     []client.GetOption
	}
	tests := []struct {
		name    string
		args    args
		want    coap.DetailedResponse[interface{}]
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceID: d1.GetID().String(),
				href:     device.ResourceURI,
				opts: []client.GetOption{
					client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration()),
				},
			},
			want: coap.DetailedResponse[interface{}]{
				Code: codes.Content,
				Body: map[interface{}]interface{}{
					"di":   d1.GetID().String(),
					"piid": d1.GetProtocolIndependentID().String(),
					"n":    d1.GetName(),
				},
			},
		},
		{
			name: "valid with interface",
			args: args{
				deviceID: d2.GetID().String(),
				href:     device.ResourceURI,
				opts: []client.GetOption{
					client.WithInterface(interfaces.OC_IF_BASELINE),
				},
			},

			want: coap.DetailedResponse[interface{}]{
				Code: codes.Content,
				Body: map[interface{}]interface{}{
					"di":   d2.GetID().String(),
					"piid": d2.GetProtocolIndependentID().String(),
					"n":    d2.GetName(),
					"if":   []interface{}{interfaces.OC_IF_BASELINE, interfaces.OC_IF_R},
					"rt":   []interface{}{"oic.d.virtual", "oic.wk.d"},
				},
			},
		},
		{
			name: "invalid href",
			args: args{
				deviceID: d1.GetID().String(),
				href:     "/invalid/href",
			},
			wantErr: true,
		},
		{
			name: "invalid deviceID",
			args: args{
				deviceID: "notfound",
				href:     device.ResourceURI,
			},
			wantErr: true,
		},
		{
			name: "invalid get handler",
			args: args{
				deviceID: d1.GetID().String(),
				href:     failRes.GetHref(),
			},
			wantErr: true,
		},
	}

	c, err := testClient.NewTestSecureClientWithBridgeSupport()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
			defer cancel()
			var got coap.DetailedResponse[interface{}]
			err := c.GetResource(ctx, tt.args.deviceID, tt.args.href, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got.ETag = nil
			require.Equal(t, tt.want, got)
		})
	}
}
