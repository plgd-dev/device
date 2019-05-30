package local_test

import (
	"context"
	"sync"
	"testing"
	"time"

	ocf "github.com/go-ocf/sdk/local"
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
				url: "c",
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

	testCfg.Protocol = "udp"
	testCfg.Resource.DiscoveryTimeout = time.Second * 3

	c, err := ocf.NewClientFromConfig(testCfg, nil)
	require := require.New(t)
	require.NoError(err)

	timeout, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	h := testOnboardDeviceHandler{}
	err = c.GetDevices(timeout, []string{"oic.d.cloudDevice"}, &h)
	require.NoError(err)
	deviceIds := h.PopDeviceIds()
	require.NotEmpty(deviceIds)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for deviceId, _ := range deviceIds {
				timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				err := c.OnboardDevice(timeout, deviceId, tt.args.authorizationProvider, tt.args.authorizationCode, tt.args.url)
				cancel()
				if tt.wantErr {
					require.Error(err)
				} else {
					require.NoError(err)
				}

			}
		})
	}
}
