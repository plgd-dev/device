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
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/require"
)

func TestClientOwnDevice(t *testing.T) {
	_ = test.MustFindDeviceByName(test.DevsimName)
	type args struct {
		deviceName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceName: test.DevsimName,
			},
		},
	}

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
			defer cancel()
			deviceID, err := test.FindDeviceByName(ctx, tt.args.deviceName)
			require.NoError(t, err)
			deviceID, err = c.OwnDevice(ctx, deviceID, client.WithOTMs([]client.OTMType{client.OTMType_JustWorks}))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			device1, err := c.GetDeviceDetailsByMulticast(ctx, deviceID)
			require.NoError(t, err)
			require.Equal(t, device1.OwnershipStatus, client.OwnershipStatus_Owned)
			err = c.DisownDevice(ctx, deviceID)
			require.NoError(t, err)
			_ = test.MustFindDeviceByName(tt.args.deviceName)
			deviceID, err = test.FindDeviceByName(ctx, tt.args.deviceName)
			require.NoError(t, err)
			deviceID, err = c.OwnDevice(ctx, deviceID, client.WithOTMs([]client.OTMType{client.OTMType_JustWorks, client.OTMType_Manufacturer}))
			require.NoError(t, err)
			device2, err := c.GetDeviceDetailsByMulticast(ctx, deviceID)
			require.NoError(t, err)
			require.Equal(t, device1.Details.(*device.Device).ProtocolIndependentID, device2.Details.(*device.Device).ProtocolIndependentID)
			require.Equal(t, device1.OwnershipStatus, client.OwnershipStatus_Owned)
			err = c.DisownDevice(ctx, deviceID)
			require.NoError(t, err)
		})
	}
}
