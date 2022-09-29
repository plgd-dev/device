package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/test"
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

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	deviceID, err = c.OwnDevice(ctx, deviceID)
	require.NoError(t, err)
	defer disown(t, c, deviceID)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, time.Second*2)
			defer cancel()
			err = c.OnboardDevice(ctx, tt.args.deviceID, tt.args.authorizationProvider, tt.args.cloudURL, tt.args.authorizationCode, tt.args.cloudID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			err = c.OffboardDevice(ctx, tt.args.deviceID)
			require.NoError(t, err)
		})
	}
}
