package local_test

import (
	"context"
	"testing"
	"time"

	grpcTest "github.com/go-ocf/grpc-gateway/test"
	"github.com/go-ocf/sdk/test"
	"github.com/stretchr/testify/require"
)

func TestClient_OwnDevice(t *testing.T) {
	secureDeviceID := grpcTest.MustFindDeviceByName(test.TestSecureDeviceName)
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
				deviceID: secureDeviceID,
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
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			err := c.OwnDevice(ctx, tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			err = c.DisownDevice(ctx, tt.args.deviceID)
			require.NoError(t, err)
		})
	}

}
