package local_test

import (
	"context"
	"testing"
	"time"

	grpcTest "github.com/go-ocf/cloud/grpc-gateway/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_OnboardDevice(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(TestDeviceName)
	type args struct {
		token                 string
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
				cloudURL:              "coap+tcp://test:5684",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			err := c.OnboardDevice(ctx, tt.args.deviceID, tt.args.authorizationProvider, tt.args.cloudURL, tt.args.authorizationCode, tt.args.cloudID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				err = c.OffboardDevice(ctx, tt.args.deviceID)
				assert.NoError(t, err)
			}
		})
	}
}
