package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/require"
)

func TestClientOwnDevice(t *testing.T) {
	_ = test.MustFindDeviceByName(test.DevsimName)
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
				deviceName: test.DevsimName,
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
			require.Equal(t, device1.OwnershipStatus, client.OwnershipStatus_Owned)
			err = c.DisownDevice(ctx, deviceID)
			require.NoError(t, err)
			_ = test.MustFindDeviceByName(tt.args.deviceName)
			deviceID, err = test.FindDeviceByName(ctx, tt.args.deviceName)
			require.NoError(t, err)
			deviceID, err = c.OwnDevice(ctx, deviceID)
			require.NoError(t, err)
			device2, err := c.GetDeviceByMulticast(ctx, deviceID)
			require.NoError(t, err)
			require.Equal(t, device1.Details.(*device.Device).ProtocolIndependentID, device2.Details.(*device.Device).ProtocolIndependentID)
			require.Equal(t, device1.OwnershipStatus, client.OwnershipStatus_Owned)
			err = c.DisownDevice(ctx, deviceID)
			require.NoError(t, err)
		})
	}
}
