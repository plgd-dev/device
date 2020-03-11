package local_test

import (
	"context"
	"testing"
	"time"

	ocf "github.com/go-ocf/sdk/local"
	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/test"
	"github.com/stretchr/testify/require"
)

func sepEpToLinks(t *testing.T, links schema.ResourceLinks) schema.ResourceLinks {
	dlink, err := ocf.GetResourceLink(links, "/oic/d")
	require.NoError(t, err)
	updateLinks := make(schema.ResourceLinks, 0, len(links))
	for _, l := range links {
		if len(l.Endpoints) == 0 {
			l.Endpoints = dlink.Endpoints
		}
		updateLinks = append(updateLinks, l)
	}
	return updateLinks
}

func TestDevice_Reboot(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "valid - iotivity-lite doesn't support reboot",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewTestSecureClient()
			require.NoError(t, err)
			defer c.Close()
			deviceId := test.TestDeviceID
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelTimeout()
			device, links, err := c.GetDevice(timeout, deviceId)
			require.NoError(err)
			defer device.Close(timeout)

			links = sepEpToLinks(t, links)

			err = device.Reboot(timeout, links)
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}

func TestDevice_FactoryReset(t *testing.T) {
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
			deviceId := testGetDeviceID(t, c.Client, false)
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelTimeout()
			device, links, err := c.GetDevice(timeout, deviceId)
			require.NoError(err)
			defer device.Close(timeout)

			links = sepEpToLinks(t, links)

			err = device.FactoryReset(timeout, links)
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}
