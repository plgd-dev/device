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
	"testing"

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/require"
)

func TestClientGetResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	type args struct {
		deviceID string
		href     string
		opts     []client.GetOption
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				opts:     []client.GetOption{client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration())},
			},
			want: map[string]interface{}{
				"n": test.DevsimName,
			},
		},
		{
			name: "valid with interface",
			args: args{
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				opts:     []client.GetOption{client.WithInterface(interfaces.OC_IF_BASELINE)},
			},
			wantErr: false,
			want: map[string]interface{}{
				"if": []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
				"n":  test.DevsimName,
				"rt": []interface{}{configuration.ResourceType},
			},
		},
		{
			name: "invalid href",
			args: args{
				deviceID: deviceID,
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
	}

	c, err := NewTestSecureClient()
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]interface{}
			err := c.GetResource(ctx, tt.args.deviceID, tt.args.href, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestClientGetDiscoveryResourceWithResourceTypeFilter(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout*8)
	defer cancel()
	deviceID, err = c.OwnDevice(ctx, deviceID)
	require.NoError(t, err)
	defer disown(t, c, deviceID)

	var v schema.ResourceLinks
	err = c.GetResource(ctx, deviceID, "/oic/res", &v, client.WithResourceTypes("oic.wk.res", "oic.wk.d"))
	require.NoError(t, err)
	require.Len(t, v, 2)
	v.Sort()
	v = cleanUpResources(v)
	require.Equal(t, test.TestDevsimResources.GetResourceLinks("oic.wk.res", "oic.wk.d").Sort(), v)
}
