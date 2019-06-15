package local_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-ocf/sdk/local/device"
	"github.com/stretchr/testify/require"
)

type testOnboardDeviceHandler struct {
	lock      sync.Mutex
	deviceIds map[string]bool
}

func (h *testOnboardDeviceHandler) Handle(ctx context.Context, client *device.Client) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.deviceIds == nil {
		h.deviceIds = make(map[string]bool)
	}
	h.deviceIds[client.DeviceID()] = true
}

func (h *testOnboardDeviceHandler) PopDeviceIds() map[string]bool {
	h.lock.Lock()
	defer h.lock.Unlock()
	tmp := h.deviceIds
	h.deviceIds = nil
	return tmp
}

func (h *testOnboardDeviceHandler) Error(err error) {
}

func TestClient_OnboardDevice(t *testing.T) {
	type args struct {
		authorizationProvider string
		authorizationCode     string
		url                   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid authorizationProvider",
			args: args{
				authorizationCode: "b",
				url:               "c",
			},
			wantErr: true,
		},
		{
			name: "invalid authorizationCode",
			args: args{
				authorizationProvider: "a",
				url:                   "c",
			},
			wantErr: true,
		},
		{
			name: "invalid url",
			args: args{
				authorizationProvider: "a",
				authorizationCode:     "b",
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				authorizationProvider: "a",
				authorizationCode:     "b",
				url:                   "c",
			},
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
		func() {
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancelTimeout()
			err := c.OwnDevice(timeout, deviceId, otm)
			require.NoError(err)

			defer func() {
				err = c.DisownDevice(timeout, deviceId)
				require.NoError(err)
			}()

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					err = c.OnboardDevice(timeout, deviceId, tt.args.authorizationProvider, tt.args.authorizationCode, tt.args.url)
					if tt.wantErr {
						require.Error(err)
					} else {
						require.NoError(err)
					}
				})
			}
		}()
	}

}
