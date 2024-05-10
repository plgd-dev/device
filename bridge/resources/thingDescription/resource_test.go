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
	"fmt"
	"net/url"
	"testing"
	"time"

	v2json "github.com/go-json-experiment/json"
	"github.com/google/uuid"
	bridgeDeviceTD "github.com/plgd-dev/device/v2/bridge/device/thingDescription"
	thingDescriptionResource "github.com/plgd-dev/device/v2/bridge/resources/thingDescription"
	"github.com/plgd-dev/device/v2/bridge/service"
	bridgeTest "github.com/plgd-dev/device/v2/bridge/test"
	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/pkg/codec/json"
	"github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/stretchr/testify/require"
	wotTD "github.com/web-of-things-open-source/thingdescription-go/thingDescription"
)

type JSONCodec struct{}

func (JSONCodec) ContentFormat() message.MediaType { return message.AppJSON }

func (JSONCodec) Encode(v interface{}) ([]byte, error) {
	return json.Encode(v)
}

func errUnknownContentFormat(err error) error {
	return fmt.Errorf("%w: %w", ocf.ErrUnknownContentFormat, err)
}

func (JSONCodec) Decode(m *pool.Message, v interface{}) error {
	if v == nil {
		return nil
	}
	mt, err := m.Options().ContentFormat()
	if err != nil {
		return errUnknownContentFormat(err)
	}
	if mt != message.AppJSON {
		return fmt.Errorf("not a JSON content format: %v", mt)
	}
	if m.Body() == nil {
		return ocf.ErrEmptyBody
	}
	if err := json.ReadFrom(m.Body(), v); err != nil {
		p, _ := m.Options().Path()
		return fmt.Errorf("decoding failed for the message %v on %v", m.Token(), p)
	}
	return nil
}

func getThingDescription(t *testing.T, data interface{}) wotTD.ThingDescription {
	tdMap, ok := data.(map[interface{}]interface{})
	require.True(t, ok)
	jsonData, err := v2json.Marshal(tdMap)
	require.NoError(t, err)
	td := wotTD.ThingDescription{}
	err = v2json.Unmarshal(jsonData, &td)
	require.NoError(t, err)
	return td
}

func getEndpoint(t *testing.T, c *client.Client, deviceID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	devices, err := c.GetDevicesDetails(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, devices[deviceID])
	eps := devices[deviceID].Endpoints
	require.NotEmpty(t, eps)
	return eps[0].URI
}

func getPatchedTD(td wotTD.ThingDescription, d service.Device, epURI string) wotTD.ThingDescription {
	return bridgeDeviceTD.PatchThingDescription(td, d, epURI, func(resourceHref string, resource bridgeDeviceTD.Resource) (wotTD.PropertyElement, bool) {
		return bridgeTest.GetPropertyElement(td, d, epURI, resourceHref, resource, message.AppCBOR)
	})
}

func TestGetThingDescription(t *testing.T) {
	s := bridgeTest.NewBridgeService(t)
	t.Cleanup(func() {
		_ = s.Shutdown()
	})
	deviceID := uuid.New()
	d := bridgeTest.NewBridgedDevice(t, s, deviceID.String(), true, true, true)
	defer func() {
		s.DeleteAndCloseDevice(d.GetID())
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

	td, err := bridgeTest.ThingDescription(deviceID, "", true, true)
	require.NoError(t, err)
	epURI := getEndpoint(t, c, d.GetID().String())
	td = getPatchedTD(td, d, epURI)

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
			name: "cbor",
			args: args{
				deviceID: d.GetID().String(),
				href:     thingDescriptionResource.ResourceURI,
			},
			want: td,
		},
		{
			name: "json",
			args: args{
				deviceID: d.GetID().String(),
				href:     thingDescriptionResource.ResourceURI,
				opts: []client.GetOption{
					client.WithCodec(JSONCodec{}),
				},
			},
			want: td,
		},
		{
			name: "json",
			args: args{
				deviceID: d.GetID().String(),
				href:     thingDescriptionResource.ResourceURI,
				opts: []client.GetOption{
					client.WithCodec(ocf.RawCodec{
						EncodeMediaType:  message.TextPlain,
						DecodeMediaTypes: []message.MediaType{message.TextPlain},
					}),
				},
			},
			wantErr: true,
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

func TestObserveThingDescription(t *testing.T) {
	s := bridgeTest.NewBridgeService(t)
	t.Cleanup(func() {
		_ = s.Shutdown()
	})

	deviceID := uuid.New()
	td, err := bridgeTest.ThingDescription(deviceID, "", true, true)
	require.NoError(t, err)
	d := bridgeTest.NewBridgedDeviceWithThingDescription(t, s, deviceID.String(), true, true, &td)
	defer func() {
		s.DeleteAndCloseDevice(d.GetID())
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

	epURI := getEndpoint(t, c, d.GetID().String())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*8)
	defer cancel()

	h := testClient.MakeMockResourceObservationHandler()
	obsID, err := c.ObserveResource(ctx, d.GetID().String(), thingDescriptionResource.ResourceURI, h)
	require.NoError(t, err)
	defer func() {
		_, errC := c.StopObservingResource(ctx, obsID)
		require.NoError(t, errC)
	}()

	n, err := h.WaitForNotification(ctx)
	require.NoError(t, err)
	var originalResource map[interface{}]interface{}
	err = n(&originalResource)
	require.NoError(t, err)
	require.Equal(t, getPatchedTD(td, d, epURI), getThingDescription(t, originalResource))

	base, err := url.Parse("http://localhost:8080")
	require.NoError(t, err)
	id, err := bridgeDeviceTD.GetThingDescriptionID(deviceID.String())
	require.NoError(t, err)
	td2 := wotTD.ThingDescription{
		Base:                *base,
		ID:                  id,
		SecurityDefinitions: map[string]wotTD.SecurityScheme{},
	}
	d.GetThingDescriptionManager().NotifySubscriptions(td2)
	n, err = h.WaitForNotification(ctx)
	require.NoError(t, err)
	var changedResource map[interface{}]interface{}
	err = n(&changedResource)
	require.NoError(t, err)
	require.Equal(t, td2, getThingDescription(t, changedResource))
}
