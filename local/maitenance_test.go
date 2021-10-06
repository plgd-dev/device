package local_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/sdk/v2/test"
	"github.com/stretchr/testify/require"
)

func TestClient_FactoryReset(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)

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
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
			defer cancel()
			err := c.FactoryReset(ctx, tt.args.deviceID)
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
	deviceID = test.MustFindDeviceByName(test.TestDeviceName)
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
		err := c.Close(context.Background())
		require.NoError(t, err)
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
				time.Sleep(time.Second * 8) // restart devsim takes arround 8seconds
			}
		})
	}
}
*/
