package local_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDevice_GetResourceLinks(t *testing.T) {
	type args struct {
		secure bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "secure",
			args: args{
				secure: true,
			},
		},
		{
			name: "insecure",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := setupSecureClient(t)
			deviceId := testGetDeviceID(t, c, tt.args.secure)
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelTimeout()

			device, err := c.GetDevice(timeout, deviceId)
			require.NoError(err)
			defer device.Close(timeout)

			got, err := device.GetResourceLinks(timeout)
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
				require.NotEmpty(got)
			}
		})
	}
}
