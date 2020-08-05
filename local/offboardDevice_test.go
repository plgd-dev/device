package local_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-ocf/sdk/test"
	"github.com/stretchr/testify/require"
)

func TestClient_OffboardDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestSecureDeviceName)
	type args struct {
		token    string
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
			wantErr: true,
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
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			err := c.OffboardDevice(ctx, tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
