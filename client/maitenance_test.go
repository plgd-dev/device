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
	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/stretchr/testify/require"
)

func TestClientFactoryReset(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)

	type args struct {
		deviceID string
		opts     []client.CommonCommandOption
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
				opts:     []client.CommonCommandOption{client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration())},
			},
		},
		{
			name: "not found",
			args: args{
				deviceID: "notFound",
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
	_, err = c.OwnDevice(ctx, deviceID)
	require.NoError(t, err)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.FactoryReset(ctx, tt.args.deviceID, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

/* TODO: not supported by iotivity-lite devsim
func TestClient_Reboot(t *testing.T) {
	deviceID = test.MustFindDeviceByName(test.DevsimName)
	type args struct {
		deviceID string
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
			},
		},
		{
			name: "not found",
			args: args{
				deviceID: "notFound",
			},
			wantErr: true,
		},
	}

	c := NewTestClient()
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
			defer cancel()
			err := c.Reboot(ctx, tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				time.Sleep(time.Second * 8) // restart devsim takes around 8seconds
			}
		})
	}
}
*/
