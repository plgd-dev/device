package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/require"
)

func TestDeviceGetEndpoints(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewTestSecureClient()
			require.NoError(t, err)
			defer func() {
				errClose := c.Close()
				require.NoError(t, errClose)
			}()
			deviceID := tt.args.deviceID
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancelTimeout()

			device, err := c.GetDeviceByMulticast(timeout, deviceID, core.DefaultDiscoveryConfiguration())
			require.NoError(err)
			defer func() {
				errClose := device.Close(timeout)
				require.NoError(errClose)
			}()

			got := device.GetEndpoints()
			require.NotEmpty(got)
		})
	}
}
