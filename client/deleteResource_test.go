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
	"strconv"
	"testing"

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/stretchr/testify/require"
)

func createSwitches(ctx context.Context, t *testing.T, c *client.Client, deviceID string, n int) {
	for i := 1; i <= n; i++ {
		var got map[string]interface{}
		err := c.CreateResource(ctx, deviceID, test.TestResourceSwitchesHref, test.MakeSwitchResourceDefaultData(), &got)
		require.NoError(t, err)
		delete(got, "ins")
		require.Equal(t, test.MakeSwitchResourceData(map[string]interface{}{
			"href": test.TestResourceSwitchesInstanceHref(strconv.Itoa(i)),
			"rep": map[interface{}]interface{}{
				"if":    []interface{}{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
				"rt":    []interface{}{types.BINARY_SWITCH},
				"value": false,
			},
		}), got)
	}
}

func TestClientDeleteResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	const switchID = "1"
	type args struct {
		deviceID string
		href     string
		opts     []client.DeleteOption
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceID: deviceID,
				href:     test.TestResourceSwitchesInstanceHref(switchID),
				opts:     []client.DeleteOption{client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration())},
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

	c, err := testClient.NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
	defer cancel()
	deviceID, err = c.OwnDevice(ctx, deviceID, client.WithOTMs([]client.OTMType{client.OTMType_JustWorks}))
	require.NoError(t, err)
	defer disown(t, c, deviceID)

	createSwitches(ctx, t, c, deviceID, 1)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.DeleteResource(ctx, tt.args.deviceID, tt.args.href, nil, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestClientBatchDeleteResources(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)

	c, err := testClient.NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
	defer cancel()
	deviceID, err = c.OwnDevice(ctx, deviceID, client.WithOTMs([]client.OTMType{client.OTMType_JustWorks}))
	require.NoError(t, err)
	defer disown(t, c, deviceID)

	createSwitches(ctx, t, c, deviceID, 8)

	switches := schema.ResourceLinks{}
	err = c.GetResource(ctx, deviceID, resources.ResourceURI, &switches, client.WithResourceTypes(types.BINARY_SWITCH))
	require.NoError(t, err)
	require.Len(t, switches, 8)

	var resp interface{}
	err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesHref, &resp, client.WithInterface(interfaces.OC_IF_B))
	require.NoError(t, err)

	switches = schema.ResourceLinks{}
	err = c.GetResource(ctx, deviceID, resources.ResourceURI, &switches, client.WithResourceTypes(types.BINARY_SWITCH))
	require.Error(t, err)
}
