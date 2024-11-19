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
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/stretchr/testify/require"
)

func TestClientUpdateResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
	defer cancel()

	c, err := testClient.NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	deviceID, err = c.OwnDevice(ctx, deviceID)
	require.NoError(t, err)
	defer disown(t, c, deviceID)

	var nonDiscoverableResource map[string]interface{}
	err = c.CreateResource(ctx, deviceID, test.TestResourceSwitchesHref, test.MakeNonDiscoverableSwitchData(), &nonDiscoverableResource)
	require.NoError(t, err)

	type args struct {
		deviceID string
		href     string
		data     interface{}
		opts     []client.UpdateOption
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
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				data: map[string]interface{}{
					"n": t.Name() + "-valid",
				},
				opts: []client.UpdateOption{client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration())},
			},
			want: coap.DetailedResponse[interface{}]{
				Code: codes.Changed,
				Body: map[interface{}]interface{}{
					"n": t.Name() + "-valid",
				},
			},
		},
		{
			name: "valid with interface",
			args: args{
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				data: map[string]interface{}{
					"n": t.Name() + "-valid with interface",
				},
				opts: []client.UpdateOption{client.WithInterface(interfaces.OC_IF_BASELINE)},
			},
			want: coap.DetailedResponse[interface{}]{
				Code: codes.Changed,
				Body: map[interface{}]interface{}{
					"n": t.Name() + "-valid with interface",
				},
			},
		},
		{
			name: "update non-discoverable resource",
			args: args{
				deviceID: deviceID,
				href:     nonDiscoverableResource["href"].(string),
				data: map[string]interface{}{
					"value": true,
				},
				opts: []client.UpdateOption{
					client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration()),
					// create the link for non-discoverable resource by utilizing the linkNotFoundCallback
					// as the only thing that we need in the link is the href and endpoints we will reuse
					// some known discoverable resource
					client.WithLinkNotFoundCallback(func(links schema.ResourceLinks, href string) (schema.ResourceLink, error) {
						resourceLink, _ := links.GetResourceLink(configuration.ResourceURI)
						resourceLink.Href = href
						return resourceLink, nil
					}),
				},
			},
			want: coap.DetailedResponse[interface{}]{
				Code: codes.Changed,
				Body: map[interface{}]interface{}{
					"value": true,
				},
			},
		},
		{
			name: "valid - revert update",
			args: args{
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				data: map[string]interface{}{
					"n": test.DevsimName,
				},
			},
			want: coap.DetailedResponse[interface{}]{
				Code: codes.Changed,
				Body: map[interface{}]interface{}{
					"n": test.DevsimName,
				},
			},
		},
		{
			name: "invalid href",
			args: args{
				deviceID: deviceID,
				href:     "/invalid/href",
				data: map[string]interface{}{
					"n": "devsim",
				},
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got coap.DetailedResponse[interface{}]
			err = c.UpdateResource(ctx, tt.args.deviceID, tt.args.href, tt.args.data, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestClientUpdateResourceInRFOTM(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	type args struct {
		deviceID string
		href     string
		data     interface{}
		opts     []client.UpdateOption
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
				deviceID: deviceID,
				href:     test.TestResourceLightInstanceHref("1"),
				data: map[string]interface{}{
					"state": true,
				},
				opts: []client.UpdateOption{client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration())},
			},
			want: coap.DetailedResponse[interface{}]{
				Code: codes.Content,
				Body: map[interface{}]interface{}{
					"name":  "Light",
					"power": uint64(0),
					"state": true,
				},
			},
		},
		{
			name: "valid with interface",
			args: args{
				deviceID: deviceID,
				href:     test.TestResourceLightInstanceHref("1"),
				data: map[string]interface{}{
					"power": 1,
				},
				opts: []client.UpdateOption{client.WithInterface(interfaces.OC_IF_BASELINE)},
			},
			want: coap.DetailedResponse[interface{}]{
				Code: codes.Content,
				Body: map[interface{}]interface{}{
					"name":  "Light",
					"power": uint64(1),
					"state": true,
				},
			},
		},
		{
			name: "valid - revert update",
			args: args{
				deviceID: deviceID,
				href:     test.TestResourceLightInstanceHref("1"),
				data: map[string]interface{}{
					"state": false,
					"power": uint64(0),
				},
			},
			want: coap.DetailedResponse[interface{}]{
				Code: codes.Content,
				Body: map[interface{}]interface{}{
					"name":  "Light",
					"power": uint64(0),
					"state": false,
				},
			},
		},
		{
			name: "forbidden",
			args: args{
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				data: map[string]interface{}{
					"n": t.Name() + "-forbidden",
				},
				opts: []client.UpdateOption{client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration())},
			},
			wantErr: true,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
	defer cancel()

	c, err := testClient.NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	_, links, err := c.GetDevice(ctx, deviceID)
	require.NoError(t, err)
	l, ok := links.GetResourceLink(test.TestResourceLightInstanceHref("1"))
	if !ok {
		t.Skip("Device doesn't support light resource")
	}
	if len(l.GetUnsecureEndpoints()) == 0 {
		t.Skip("Device doesn't support access to light resource via unsecure endpoint")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = c.UpdateResource(ctx, tt.args.deviceID, tt.args.href, tt.args.data, nil, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			var got coap.DetailedResponse[interface{}]
			err = c.GetResource(ctx, tt.args.deviceID, tt.args.href, &got)
			require.NoError(t, err)
			got.ETag = nil
			require.Equal(t, tt.want, got)
		})
	}
}
