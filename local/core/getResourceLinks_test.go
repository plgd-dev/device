package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/sdk/v2/local/core"
	"github.com/plgd-dev/sdk/v2/test"

	"github.com/stretchr/testify/require"
)

func TestDevice_GetResourceLinks(t *testing.T) {
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
			deviceID := tt.args.deviceID
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancelTimeout()

			device, err := c.GetDeviceByMulticast(timeout, deviceID, core.DefaultDiscoveryConfiguration())
			require.NoError(err)
			defer device.Close(timeout)
			eps := device.GetEndpoints()
			links, err := device.GetResourceLinks(timeout, eps)
			require.NoError(err)

			dlink, err := core.GetResourceLink(links, "/oic/d")
			require.NoError(err)
			got, err := device.GetResourceLinks(timeout, dlink.GetEndpoints())
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
				require.NotEmpty(got)
			}
		})
	}
}
