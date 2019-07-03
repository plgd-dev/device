package local_test

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"sync"
	"testing"
	"time"

	"github.com/go-ocf/sdk/local"
	ocf "github.com/go-ocf/sdk/local"
	"github.com/go-ocf/sdk/schema"
	"github.com/stretchr/testify/require"
)

type testFindDeviceHandler struct {
	secured bool

	lock      sync.Mutex
	deviceIds map[string]bool
}

func (h *testFindDeviceHandler) Handle(ctx context.Context, d *local.Device) {
	if !d.GetDeviceLinks().IsSecured() == h.secured {
		return
	}
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.deviceIds == nil {
		h.deviceIds = make(map[string]bool)
	}
	h.deviceIds[d.DeviceID()] = true
}

func (h *testFindDeviceHandler) PopDeviceIds() map[string]bool {
	h.lock.Lock()
	defer h.lock.Unlock()
	tmp := h.deviceIds
	h.deviceIds = nil
	return tmp
}

func (h *testFindDeviceHandler) DeviceIDs() []string {
	h.lock.Lock()
	defer h.lock.Unlock()

	out := make([]string, 0, len(h.deviceIds))
	for id, _ := range h.deviceIds {
		out = append(out, id)
	}
	return out
}

func (h *testFindDeviceHandler) Error(err error) {
}

func testGetDeviceID(t *testing.T, c *ocf.Client, secured bool) string {
	timeout, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	h := testFindDeviceHandler{secured: secured}
	err := c.GetDevices(timeout, []string{"oic.d.cloudDevice"}, &h)
	require.NoError(t, err)
	deviceIds := h.PopDeviceIds()
	require.NotEmpty(t, deviceIds)
	require.Len(t, deviceIds, 1)
	for key, _ := range deviceIds {
		return key
	}
	return ""
}

func testGetProvisionDevice(t *testing.T) ocf.ProvisionDeviceFunc {
	return func(ctx context.Context, c *ocf.ProvisioningClient) error {
		derBlock, _ := pem.Decode(CARootPemBlock)
		require.NotEmpty(t, derBlock)
		ca, err := x509.ParseCertificate(derBlock.Bytes)
		require.NoError(t, err)

		err = c.AddCertificateAuthority(ctx, "*", ca)
		if err != nil {
			return err
		}
		return c.SetCloudResource(ctx, schema.CloudUpdateRequest{
			AuthorizationProvider: "a",
			AuthorizationCode:     "b",
			URL:                   "c",
		})
	}
}

func TestClient_OnboardDevice(t *testing.T) {

	type args struct {
		provision ocf.ProvisionDeviceFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid authorizationProvider",
			args: args{
				provision: func(ctx context.Context, c *ocf.ProvisioningClient) error {
					return c.SetCloudResource(ctx, schema.CloudUpdateRequest{})
				},
			},
			wantErr: true,
		},
		{
			name: "invalid authorizationCdode",
			args: args{
				provision: func(ctx context.Context, c *ocf.ProvisioningClient) error {
					return c.SetCloudResource(ctx, schema.CloudUpdateRequest{
						AuthorizationProvider: "a",
					})
				},
			},
			wantErr: true,
		},
		{
			name: "invalid url",
			args: args{
				provision: func(ctx context.Context, c *ocf.ProvisioningClient) error {
					return c.SetCloudResource(ctx, schema.CloudUpdateRequest{
						AuthorizationProvider: "a",
						AuthorizationCode:     "b",
					})
				},
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				provision: testGetProvisionDevice(t),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, otm := setupSecureClient(t)
			require := require.New(t)
			deviceId := testGetDeviceID(t, c, true)
			timeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			err := c.OnboardDevice(timeout, deviceId, otm, tt.args.provision)
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
				err = c.OffboardDevice(timeout, deviceId)
				require.NoError(err)
			}
		})
	}
}

func TestClient_OnboardDevice2Times(t *testing.T) {
	c, otm := setupSecureClient(t)
	require := require.New(t)

	deviceId := testGetDeviceID(t, c, true)
	timeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	p := testGetProvisionDevice(t)

	err := c.OnboardDevice(timeout, deviceId, otm, p)
	require.NoError(err)

	err = c.OnboardDevice(timeout, deviceId, otm, p)
	require.NoError(err)

	err = c.OffboardDevice(timeout, deviceId)
	require.NoError(err)
}

func TestClient_OnboardInsecureDevice(t *testing.T) {
	type args struct {
		AuthorizationProvider string
		AuthorizationCode     string
		URL                   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "invalid authorizationProvider",
			args:    args{},
			wantErr: true,
		},
		{
			name: "invalid authorizationCode",
			args: args{
				AuthorizationProvider: "a",
			},
			wantErr: true,
		},
		{
			name: "invalid url",
			args: args{
				AuthorizationProvider: "a",
				AuthorizationCode:     "b",
			},
			wantErr: true,
		},
		{
			name: "valid",
			args: args{
				AuthorizationProvider: "a",
				AuthorizationCode:     "b",
				URL:                   "c",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			cfg := testCfg
			cfg.Protocol = "udp"
			c, err := ocf.NewClientFromConfig(cfg, nil)
			require.NoError(err)
			deviceId := testGetDeviceID(t, c, false)
			timeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			err = c.OnboardInsecuredDevice(timeout, deviceId, tt.args.AuthorizationProvider, tt.args.AuthorizationCode, tt.args.URL)
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
				err = c.OffboardInsecuredDevice(timeout, deviceId)
				require.NoError(err)
			}
		})
	}
}
