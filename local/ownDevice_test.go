package local_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/sdk/local"
	"github.com/plgd-dev/sdk/schema"
	"github.com/plgd-dev/sdk/test"
	"github.com/stretchr/testify/require"
)

func TestClient_OwnDevice(t *testing.T) {
	_ = test.MustFindDeviceByName(test.TestSecureDeviceName)
	type args struct {
		deviceName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceName: test.TestSecureDeviceName,
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
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
			defer cancel()
			deviceID, err := test.FindDeviceByName(ctx, tt.args.deviceName)
			require.NoError(t, err)
			deviceID, err = c.OwnDevice(ctx, deviceID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			device1, err := c.GetDeviceByMulticast(ctx, deviceID)
			require.NoError(t, err)
			require.Equal(t, device1.OwnershipStatus, local.OwnershipStatus_Owned)
			err = c.DisownDevice(ctx, deviceID)
			require.NoError(t, err)
			_ = test.MustFindDeviceByName(test.TestSecureDeviceName)
			deviceID, err = test.FindDeviceByName(ctx, tt.args.deviceName)
			require.NoError(t, err)
			deviceID, err = c.OwnDevice(ctx, deviceID)
			require.NoError(t, err)
			device2, err := c.GetDeviceByMulticast(ctx, deviceID)
			require.NoError(t, err)
			require.Equal(t, device1.Details.(*schema.Device).ProtocolIndependentID, device2.Details.(*schema.Device).ProtocolIndependentID)
			require.Equal(t, device1.OwnershipStatus, local.OwnershipStatus_Owned)
			err = c.DisownDevice(ctx, deviceID)
			require.NoError(t, err)
		})
	}

}
