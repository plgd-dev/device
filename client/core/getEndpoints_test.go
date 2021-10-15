package core_test

import (
	"context"
	"testing"
	"time"

	ocf "github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/test"

	"github.com/stretchr/testify/require"
)

func TestDeviceGetEndpoints(t *testing.T) {
	deviceID := test.MustFindDeviceByName(TestDeviceName)
	secureDeviceID := test.MustFindDeviceByName(test.TestSecureDeviceName)
	type args struct {
		deviceID string
	}
	tests := []struct {
		name string
		args args
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
			deviceID := tt.args.deviceID
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancelTimeout()

			device, err := c.GetDeviceByMulticast(timeout, deviceID, ocf.DefaultDiscoveryConfiguration())
			require.NoError(err)
			defer device.Close(timeout)

			got := device.GetEndpoints()
			require.NotEmpty(got)
		})
	}
}
