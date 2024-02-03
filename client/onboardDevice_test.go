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

	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/stretchr/testify/require"
)

func TestClientOnboardDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	type args struct {
		deviceID              string
		authorizationProvider string
		authorizationCode     string
		cloudURL              string
		cloudID               string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceID:              deviceID,
				authorizationProvider: "authorizationProvider",
				authorizationCode:     "authorizationCode",
				cloudURL:              "coaps+tcp://test:5684",
				cloudID:               "cloudID",
			},
		},
		{
			name: "notFound",
			args: args{
				deviceID:              "notFound",
				authorizationProvider: "authorizationProvider",
				authorizationCode:     "authorizationCode",
				cloudURL:              "coaps+tcp://test:5684",
				cloudID:               "cloudID",
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
	deviceID, err = c.OwnDevice(ctx, deviceID)
	require.NoError(t, err)
	defer disown(t, c, deviceID)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttCtx, ttCancel := context.WithTimeout(ctx, time.Second*2)
			defer ttCancel()
			err = c.OnboardDevice(ttCtx, tt.args.deviceID, tt.args.authorizationProvider, tt.args.cloudURL, tt.args.authorizationCode, tt.args.cloudID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			err = c.OffboardDevice(ttCtx, tt.args.deviceID)
			require.NoError(t, err)
		})
	}
}
