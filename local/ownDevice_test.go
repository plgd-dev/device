package local_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-ocf/sdk/test"
	"github.com/stretchr/testify/require"
)

func TestClient_ownDevice(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name: "valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewTestSecureClient()
			require.NoError(t, err)
			defer c.Close()
			deviceId := test.TestSecureDeviceID
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancelTimeout()

			device, links, err := c.GetDevice(timeout, deviceId)
			require.NoError(err)
			defer device.Close(timeout)

			err = device.Own(timeout, links, c.otm)
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
			err = device.Disown(timeout, links)
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
				time.Sleep(time.Second)
				// deviceID is changed after disown
				test.TestSecureDeviceID = test.MustFindDeviceByName(test.TestSecureDeviceName)
			}
		})
	}
}
