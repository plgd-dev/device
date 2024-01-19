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

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/client/core"
	codecOcf "github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/configuration"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/introspection"
	"github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/schema/plgdtime"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/schema/softwareupdate"
	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/plgd-dev/go-coap/v3/message/codes"
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
		want    coap.DetailedResponse[interface{}]
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				opts:     []client.GetOption{client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration())},
			},
			want: coap.DetailedResponse[interface{}]{
				Code: codes.Content,
				Body: map[interface{}]interface{}{
					"n": test.DevsimName,
				},
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
			want: coap.DetailedResponse[interface{}]{
				Code: codes.Content,
				Body: map[interface{}]interface{}{
					"if": []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
					"n":  test.DevsimName,
					"rt": []interface{}{configuration.ResourceType},
				},
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
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	deviceID, err = c.OwnDevice(ctx, deviceID)
	require.NoError(t, err)
	defer disown(t, c, deviceID)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestClientGetDiscoveryResourceWithResourceTypeFilter(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := testClient.NewTestSecureClient()
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
	err = c.GetResource(ctx, deviceID, resources.ResourceURI, &v, client.WithResourceTypes("oic.wk.res", "oic.wk.d"))
	require.NoError(t, err)
	require.Len(t, v, 2)
	v.Sort()
	v = cleanUpResources(v)
	require.Equal(t, test.TestDevsimResources.GetResourceLinks(resources.ResourceType, "oic.wk.d").Sort(), v)
}

func updateConfigurationResource(ctx context.Context, c *client.Client, deviceID string) error {
	var got interface{}
	err := c.UpdateResource(ctx, deviceID, configuration.ResourceURI, map[string]interface{}{
		"n": test.DevsimName + "-updated",
	}, &got)
	if err != nil {
		return err
	}

	// restore name - for other tests following this test case
	return c.UpdateResource(ctx, deviceID, configuration.ResourceURI, map[string]interface{}{
		"n": test.DevsimName,
	}, &got)
}

func TestClientGetDiscoveryResourceWithBatchInterface(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := testClient.NewTestSecureClient()
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

	// update /oc/con, etags are always increasing so by updating /oc/con we ensure that it has the highest etag
	err = updateConfigurationResource(ctx, c, deviceID)
	require.NoError(t, err)

	d, links, err := c.GetDevice(ctx, deviceID, client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration()))
	require.NoError(t, err)
	link, err := core.GetResourceLink(links, resources.ResourceURI)
	require.NoError(t, err)
	// force the use of a secure endpoint
	link.Endpoints = link.Endpoints.FilterSecureEndpoints()

	v := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
	opts := []coap.OptionFunc{coap.WithInterface(interfaces.OC_IF_B)}
	err = d.GetResourceWithCodec(ctx, link, codecOcf.VNDOCFCBORCodec{}, &v, opts...)
	require.NoError(t, err)
	require.Equal(t, codes.Content, v.Code)
	if v.ETag == nil {
		t.Skip("Device doesn't support ETag")
	}

	for i := range v.Body {
		require.Equal(t, deviceID, v.Body[i].DeviceID())
		switch v.Body[i].Href() {
		case platform.ResourceURI:
		case plgdtime.ResourceURI:
		case configuration.ResourceURI:
		case introspection.ResourceURI:
		case maintenance.ResourceURI:
		case cloud.ResourceURI:
		case device.ResourceURI:
		case test.TestResourceLightInstanceHref("1"):
		case test.TestResourceSwitchesHref:
		case softwareupdate.ResourceURI:
		default:
			require.NoError(t, fmt.Errorf("unknown resource href: %v", v.Body[i].Href()))
		}
	}

	var etag []byte
	for _, bi := range v.Body {
		if bi.Href() == configuration.ResourceURI {
			etag = bi.ETag
		}
		require.NotEmpty(t, bi.ETag)
	}
	require.Equal(t, v.ETag, etag)

	v = coap.DetailedResponse[resources.BatchResourceDiscovery]{}
	// when using a valid ETag we should get a Valid response with an empty payload
	opts = []coap.OptionFunc{
		coap.WithInterface(interfaces.OC_IF_B),
		coap.WithETag(etag),
	}
	err = d.GetResourceWithCodec(ctx, link, codecOcf.VNDOCFCBORCodec{}, &v, opts...)
	require.NoError(t, err)
	require.Equal(t, codes.Valid, v.Code)
	require.Empty(t, v.Body)
}

// Creating and deleting resource should update the batch etag of the /oic/res resource
func TestClientGetDiscoveryResourceWithBatchInterfaceCreateAndDeleteResource(t *testing.T) {
	if !ETagBatchSupported {
		t.Skip("Device doesn't support ETag for batch interface")
	}

	deviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := testClient.NewTestSecureClient()
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

	d, links, err := c.GetDevice(ctx, deviceID, client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration()))
	require.NoError(t, err)
	link, err := core.GetResourceLink(links, resources.ResourceURI)
	require.NoError(t, err)
	// force the use of a secure endpoint
	secureEndpoints := link.Endpoints.FilterSecureEndpoints()
	if (len(secureEndpoints)) != 0 {
		link.Endpoints = secureEndpoints
	}

	// get batch etag
	getBatchEtag := func() []byte {
		v := coap.DetailedResponse[resources.BatchResourceDiscovery]{}
		opts := []coap.OptionFunc{coap.WithInterface(interfaces.OC_IF_B)}
		err = d.GetResourceWithCodec(ctx, link, codecOcf.VNDOCFCBORCodec{}, &v, opts...)
		require.NoError(t, err)
		require.Equal(t, codes.Content, v.Code)
		return v.ETag
	}
	etag1 := getBatchEtag()
	require.NotNil(t, etag1)

	// add resource
	err = c.CreateResource(ctx, deviceID, test.TestResourceSwitchesHref, test.MakeSwitchResourceDefaultData(), nil)
	require.NoError(t, err)
	etag2 := getBatchEtag()
	require.NotNil(t, etag2)
	require.NotEqual(t, etag1, etag2)

	// remove resource
	err = c.DeleteResource(ctx, deviceID, test.TestResourceSwitchesInstanceHref("1"), nil)
	require.NoError(t, err)
	etag3 := getBatchEtag()
	require.NotNil(t, etag3)
	require.NotEqual(t, etag2, etag3)
}

func TestClientGetConResourceByETag(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := testClient.NewTestSecureClient()
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

	v1 := coap.DetailedResponse[configuration.Configuration]{}
	err = c.GetResource(ctx, deviceID, configuration.ResourceURI, &v1)
	require.NoError(t, err)
	require.Equal(t, codes.Content, v1.Code)
	if v1.ETag == nil {
		t.Skip("Device doesn't support ETag")
	}
	etag := v1.ETag

	// if the resource is not changed, we should always get the same ETag
	v2 := coap.DetailedResponse[configuration.Configuration]{}
	err = c.GetResource(ctx, deviceID, configuration.ResourceURI, &v2)
	require.NoError(t, err)
	require.Equal(t, codes.Content, v2.Code)
	require.Equal(t, etag, v2.ETag)

	// non-matching ETag should return content and current ETag
	etag2 := []byte{'l', 'e', 'e', 't', '4', '2'}
	err = c.GetResource(ctx, deviceID, configuration.ResourceURI, &v1, client.WithETag(etag2))
	require.NoError(t, err)
	require.Equal(t, codes.Content, v1.Code)
	require.Equal(t, etag, v1.ETag)

	// matching ETag should return valid and no content
	v1 = coap.DetailedResponse[configuration.Configuration]{}
	err = c.GetResource(ctx, deviceID, configuration.ResourceURI, &v1, client.WithETag(etag))
	require.NoError(t, err)
	require.Equal(t, codes.Valid, v1.Code)
	require.Empty(t, v1.Body)

	// after update, ETag should change
	err = updateConfigurationResource(ctx, c, deviceID)
	require.NoError(t, err)

	// previous ETag should no longer match
	v3 := coap.DetailedResponse[configuration.Configuration]{}
	err = c.GetResource(ctx, deviceID, configuration.ResourceURI, &v3)
	require.NoError(t, err)
	require.Equal(t, codes.Content, v3.Code)
	require.NotEqual(t, etag, v3.ETag)
}
