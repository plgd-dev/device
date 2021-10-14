package core_test

import (
	"context"
	"testing"
	"time"

	ocf "github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/require"
)

func sepEpToLinks(t *testing.T, links schema.ResourceLinks) schema.ResourceLinks {
	dlink, err := ocf.GetResourceLink(links, device.ResourceURI)
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
			deviceID := test.MustFindDeviceByName(TestDeviceName)
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelTimeout()
			device, err := c.GetDeviceByMulticast(timeout, deviceID, ocf.DefaultDiscoveryConfiguration())
			require.NoError(err)
			defer device.Close(timeout)
			eps := device.GetEndpoints()
			links, err := device.GetResourceLinks(timeout, eps)
			require.NoError(err)

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
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()
	deviceID := test.MustFindDeviceByName(TestDeviceName)
	require := require.New(t)
	timeout, cancelTimeout := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelTimeout()
	device, err := c.GetDeviceByMulticast(timeout, deviceID, ocf.DefaultDiscoveryConfiguration())
	require.NoError(err)
	defer device.Close(timeout)
	eps := device.GetEndpoints()
	links, err := device.GetResourceLinks(timeout, eps)
	require.NoError(err)

	links = sepEpToLinks(t, links)

	err = device.FactoryReset(timeout, links)
	require.NoError(err)
}
