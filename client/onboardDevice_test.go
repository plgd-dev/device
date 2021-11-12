package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/require"
)

func TestClientOnboardDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimNetHost)
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
	}

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	deviceID, err = c.OwnDevice(ctx, deviceID)
	require.NoError(t, err)
	defer func() {
		err := c.DisownDevice(ctx, deviceID)
		require.NoError(t, err)
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = c.OnboardDevice(ctx, deviceID, tt.args.authorizationProvider, tt.args.cloudURL, tt.args.authorizationCode, tt.args.cloudID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				err = c.OffboardDevice(ctx, deviceID)
				require.NoError(t, err)
			}
		})
	}
}
