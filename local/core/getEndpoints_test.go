package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/sdk/test"

	"github.com/stretchr/testify/require"
)

func TestDevice_GetEndpoints(t *testing.T) {
	deviceID := test.MustFindDeviceByName(TestDeviceName)
	secureDeviceID := test.MustFindDeviceByName(test.TestSecureDeviceName)
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
		{
			name: "insecure",
			args: args{
				deviceID: deviceID,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewTestSecureClient()
			require.NoError(t, err)
			defer c.Close()
			deviceId := tt.args.deviceID
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancelTimeout()

			device, _, err := c.GetDevice(timeout, deviceId)
			require.NoError(err)
			defer device.Close(timeout)

			got, err := device.GetEndpoints(timeout)
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
				require.NotEmpty(got)
			}
		})
	}
}
