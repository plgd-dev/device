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
	"time"

	"github.com/plgd-dev/device/v2/client"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/test"
	testTypes "github.com/plgd-dev/device/v2/test/resource/types"
	"github.com/stretchr/testify/require"
)

func sortResources(s []schema.ResourceLink) []schema.ResourceLink {
	v := schema.ResourceLinks(s)
	v.Sort()
	return v
}

func NewTestDeviceSimulator(deviceID, deviceName string) client.DeviceDetails {
	return client.DeviceDetails{
		ID: deviceID,
		Details: &device.Device{
			ID:            deviceID,
			Name:          deviceName,
			ResourceTypes: []string{testTypes.DEVICE_CLOUD, device.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
		},
		Resources:       sortResources(append(test.TestDevsimResources, test.TestDevsimPrivateResources...)),
		OwnershipStatus: client.OwnershipStatus_Unknown,
	}
}

func NewTestSecureDeviceSimulator(deviceID, deviceName string, ip string) client.DeviceDetails {
	return client.DeviceDetails{
		ID: deviceID,
		Details: &device.Device{
			ID:            deviceID,
			Name:          deviceName,
			ResourceTypes: []string{testTypes.DEVICE_CLOUD, device.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_R, interfaces.OC_IF_BASELINE},
		},
		IsSecured: true,
		Ownership: &doxm.Doxm{
			ResourceOwner:                 "00000000-0000-0000-0000-000000000000",
			SupportedOwnerTransferMethods: []doxm.OwnerTransferMethod{doxm.JustWorks, doxm.ManufacturerCertificate},
			OwnerID:                       "00000000-0000-0000-0000-000000000000",
			DeviceID:                      deviceID,
			SupportedCredentialTypes:      credential.CredentialType_SYMMETRIC_PAIR_WISE | credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
			SelectedOwnerTransferMethod:   doxm.Self,
			Interfaces:                    []string{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
			ResourceTypes:                 []string{doxm.ResourceType},
		},
		Resources:       sortResources(append(append(test.TestDevsimResources, test.TestDevsimPrivateResources...), test.TestDevsimSecResources...)),
		OwnershipStatus: client.OwnershipStatus_ReadyToBeOwned,
		FoundByIP:       ip,
	}
}

func cleanUpResources(s []schema.ResourceLink) []schema.ResourceLink {
	a := make([]schema.ResourceLink, 0, len(s))
	for _, l := range s {
		l.Endpoints = nil
		l.Policy = nil
		l.Anchor = ""
		a = append(a, l)
	}
	return a
}

func TestClientGetDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	type args struct {
		deviceID string
	}
	tests := []struct {
		name    string
		args    args
		want    client.DeviceDetails
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceID: deviceID,
			},
			want: NewTestSecureDeviceSimulator(deviceID, test.DevsimName, ""),
		},
		{
			name: "not-found",
			args: args{
				deviceID: "not-found",
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel = context.WithTimeout(ctx, time.Second)
			defer cancel()
			got, err := c.GetDeviceDetailsByMulticast(ctx, tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got.Resources = cleanUpResources(sortResources(got.Resources))
			got.Endpoints = nil
			require.NotEmpty(t, got.Details.(*device.Device).ProtocolIndependentID)
			got.Details.(*device.Device).ProtocolIndependentID = ""
			require.Equal(t, tt.want, got)
		})
	}
}

func TestClientGetDeviceByIP(t *testing.T) {
	deviceIDip4 := test.MustFindDeviceByName(test.DevsimName)
	ip4 := test.MustFindDeviceIP(test.DevsimName, test.IP4)
	deviceIDip6 := test.MustFindDeviceByName(test.DevsimName)
	ip6 := test.MustFindDeviceIP(test.DevsimName, test.IP6)
	type args struct {
		ip string
	}
	tests := []struct {
		name    string
		args    args
		want    client.DeviceDetails
		wantErr bool
	}{
		{
			name: "ip4",
			args: args{
				ip: ip4,
			},
			want: NewTestSecureDeviceSimulator(deviceIDip4, test.DevsimName, ip4),
		},
		{
			name: "ip6",
			args: args{
				ip: ip6,
			},
			want: NewTestSecureDeviceSimulator(deviceIDip6, test.DevsimName, "["+ip6+"]"),
		},
		{
			name: "not-found",
			args: args{
				ip: "1.2.3.4",
			},
			wantErr: true,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel = context.WithTimeout(ctx, time.Second)
			defer cancel()
			got, err := c.GetDeviceDetailsByIP(ctx, tt.args.ip)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			var v interface{}
			err = c.GetResource(ctx, got.ID, device.ResourceURI, &v)
			require.NoError(t, err)

			got.Resources = cleanUpResources(sortResources(got.Resources))
			got.Endpoints = nil
			require.NotEmpty(t, got.Details.(*device.Device).ProtocolIndependentID)
			got.Details.(*device.Device).ProtocolIndependentID = ""
			require.Equal(t, tt.want, got)
			ok := c.DeleteDevice(ctx, got.ID)
			require.True(t, ok)

			// we should not be able to remove the device second time
			ok = c.DeleteDevice(ctx, got.ID)
			require.False(t, ok)
		})
	}
}

func TestClientCheckForDuplicityDeviceInCache(t *testing.T) {
	ip := test.MustFindDeviceIP(test.DevsimName, test.IP4)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(ctx)
		require.NoError(t, errC)
	}()
	// store device to cache
	dev, _, err := c.GetDeviceByIP(ctx, ip)
	require.NoError(t, err)
	_, err = c.OwnDevice(ctx, dev.DeviceID())
	require.NoError(t, err)
	dev, _, err = c.GetDevice(ctx, dev.DeviceID())
	require.NoError(t, err)
	err = c.DisownDevice(ctx, dev.DeviceID())
	require.NoError(t, err)
	time.Sleep(time.Second * 4)
	// deviceID was changed after disowning - the call fails, but internally the cache is updated so dev.DeviceID() returns new deviceID
	_, _, err = c.GetDevice(ctx, dev.DeviceID())
	require.Error(t, err)
	_, _, err = c.GetDevice(ctx, dev.DeviceID())
	require.NoError(t, err)

	// change deviceID by another client
	c1, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c1.Close(ctx)
		require.NoError(t, errC)
	}()
	_, err = c1.OwnDevice(ctx, dev.DeviceID())
	require.NoError(t, err)
	err = c1.DisownDevice(ctx, dev.DeviceID())
	require.NoError(t, err)
	time.Sleep(time.Second * 4)

	// try get old device again
	_, _, err = c.GetDevice(ctx, dev.DeviceID())
	require.Error(t, err)
	// dev has updated deviceID by previous call so we can get the device
	_, _, err = c.GetDevice(ctx, dev.DeviceID())
	require.NoError(t, err)
	dev.SetEndpoints(nil)
	_, _, err = c.GetDevice(ctx, dev.DeviceID())
	require.NoError(t, err)

	deletedDevices := c.DeleteDevices(ctx, []string{dev.DeviceID()})
	require.Equal(t, []string{dev.DeviceID()}, deletedDevices)

	// device is stored without cache
	dev, _, err = c.GetDevice(ctx, dev.DeviceID())
	require.NoError(t, err)
	// invalidate endpoints
	dev.SetEndpoints(nil)
	// device endpoints will be updated by multicast
	_, _, err = c.GetDevice(ctx, dev.DeviceID())
	require.NoError(t, err)
}
