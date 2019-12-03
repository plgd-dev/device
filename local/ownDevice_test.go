package local_test

import (
	"context"
	"testing"
	"time"

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
			c, err := NewTestSecureClientWithTLS(false, true)
			require.NoError(t, err)
			defer c.Close()
			deviceId := testGetDeviceID(t, c.Client, true)
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 30*time.Second)
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
			}
		})
	}
}
