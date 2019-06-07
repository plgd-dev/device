package local_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClient_OffboardDevice(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name: "valid",
		},
	}

	c, otm := setupSecureClient(t)
	require := require.New(t)

	timeout, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	h := testOnboardDeviceHandler{}
	err := c.GetDevices(timeout, []string{"oic.d.cloudDevice"}, &h)
	require.NoError(err)
	deviceIds := h.PopDeviceIds()
	require.NotEmpty(deviceIds)

	for deviceId, _ := range deviceIds {
		timeout, cancelTimeout := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelTimeout()
		err := c.OwnDevice(timeout, deviceId, otm)
		require.NoError(err)

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err = c.OffboardDevice(timeout, deviceId)
				if tt.wantErr {
					require.Error(err)
				} else {
					require.NoError(err)
				}
			})
		}

		defer func() {
			err = c.DisownDevice(timeout, deviceId)
			require.NoError(err)
		}()
	}
}
