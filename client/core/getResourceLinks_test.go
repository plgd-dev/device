package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/test"

	"github.com/stretchr/testify/require"
)

func TestDeviceGetResourceLinks(t *testing.T) {
	deviceID := test.MustFindDeviceByName(DevsimNetHost)
	secureDeviceID := test.MustFindDeviceByName(test.DevsimNetBridge)
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

			dev, err := c.GetDeviceByMulticast(timeout, deviceID, core.DefaultDiscoveryConfiguration())
			require.NoError(err)
			defer dev.Close(timeout)
			eps := dev.GetEndpoints()
			links, err := dev.GetResourceLinks(timeout, eps)
			require.NoError(err)

			dlink, err := core.GetResourceLink(links, device.ResourceURI)
			require.NoError(err)
			got, err := dev.GetResourceLinks(timeout, dlink.GetEndpoints())
			if tt.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)
			require.NotEmpty(got)
		})
	}
}
