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

package bridge_test

import (
	"context"
	"crypto/x509"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	bridgeTest "github.com/plgd-dev/device/v2/bridge/test"
	schemaCredential "github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/test"
	testClient "github.com/plgd-dev/device/v2/test/client"
	"github.com/stretchr/testify/require"
)

func TestOnboardDevice(t *testing.T) {
	s := bridgeTest.NewBridgeService(t)
	t.Cleanup(func() {
		_ = s.Shutdown()
	})
	deviceID := uuid.New().String()
	d := bridgeTest.NewBridgedDevice(t, s, deviceID, true, false)
	defer func() {
		s.DeleteAndCloseDevice(d.GetID())
	}()
	deviceIDwithoutCAPool := uuid.New().String()
	deviceWithoutCAPool := bridgeTest.NewBridgedDevice(t, s, deviceIDwithoutCAPool, true, true, device.WithCAPool(cloud.MakeCAPool(func() []*x509.Certificate {
		return nil
	}, false)))
	defer func() {
		s.DeleteAndCloseDevice(deviceWithoutCAPool.GetID())
	}()
	cleanup := bridgeTest.RunBridgeService(s)
	defer func() {
		errC := cleanup()
		require.NoError(t, errC)
	}()

	type args struct {
		deviceID              string
		authorizationProvider string
		authorizationCode     string
		cloudURL              string
		cloudID               string
		cloudCA               []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
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
			name: "validWithCA",
			args: args{
				deviceID:              deviceIDwithoutCAPool,
				authorizationProvider: "authorizationProvider",
				authorizationCode:     "authorizationCode",
				cloudURL:              "coaps+tcp://test:5684",
				cloudID:               "cloudID",
				cloudCA:               test.GetRootCApem(t),
			},
		},
	}

	c, err := testClient.NewTestSecureClientWithBridgeSupport()
	require.NoError(t, err)
	defer func() {
		errC := c.Close(context.Background())
		require.NoError(t, errC)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), test.TestTimeout)
	defer cancel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ttCtx, ttCancel := context.WithTimeout(ctx, time.Second)
			defer ttCancel()
			if len(tt.args.cloudCA) > 0 {
				err = c.UpdateResource(ctx, tt.args.deviceID, schemaCredential.ResourceURI, schemaCredential.CredentialUpdateRequest{
					Credentials: []schemaCredential.Credential{
						{
							Subject: tt.args.cloudID,
							Type:    schemaCredential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
							Usage:   schemaCredential.CredentialUsage_TRUST_CA,
							PublicData: &schemaCredential.CredentialPublicData{
								DataInternal: tt.args.cloudCA,
								Encoding:     schemaCredential.CredentialPublicDataEncoding_PEM,
							},
						},
					},
				}, nil)
				require.NoError(t, err)
				var res schemaCredential.CredentialResponse
				err = c.GetResource(ctx, tt.args.deviceID, schemaCredential.ResourceURI, &res)
				require.NoError(t, err)
				require.Len(t, res.Credentials, 1)
			}
			err = c.OnboardDevice(ttCtx, tt.args.deviceID, tt.args.authorizationProvider, tt.args.cloudURL, tt.args.authorizationCode, tt.args.cloudID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			err = c.OffboardDevice(ttCtx, tt.args.deviceID)
			require.NoError(t, err)
			if len(tt.args.cloudCA) > 0 {
				// remove CA is async
				time.Sleep(time.Second)
				var res schemaCredential.CredentialResponse
				err = c.GetResource(ctx, tt.args.deviceID, schemaCredential.ResourceURI, &res)
				require.NoError(t, err)
				require.Len(t, res.Credentials, 0)
			}
		})
	}
}
