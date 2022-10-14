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

package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/require"
)

func TestDeviceGetResourceLinks(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)
	type args struct {
		deviceID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "secure",
			args: args{
				deviceID: secureDeviceID,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewTestSecureClient()
			require.NoError(t, err)
			defer func() {
				errClose := c.Close()
				require.NoError(t, errClose)
			}()
			deviceID := tt.args.deviceID
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancelTimeout()

			dev, err := c.GetDeviceByMulticast(timeout, deviceID, core.DefaultDiscoveryConfiguration())
			require.NoError(err)
			defer func() {
				errClose := dev.Close(timeout)
				require.NoError(errClose)
			}()
			eps := dev.GetEndpoints()
			links, err := dev.GetResourceLinks(timeout, eps)
			require.NoError(err)

			dlink, err := core.GetResourceLink(links, device.ResourceURI)
			require.NoError(err)
			got, err := dev.GetResourceLinks(timeout, dlink.GetEndpoints())
			if tt.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)
			require.NotEmpty(got)
		})
	}
}
