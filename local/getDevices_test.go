package local_test

import (
	"context"
	"testing"
	"time"

	ocf "github.com/go-ocf/sdk/local"
	"github.com/go-ocf/sdk/local/device"
	"github.com/go-ocf/sdk/resource/types"

	"github.com/stretchr/testify/require"
)

func TestDeviceDiscovery(t *testing.T) {
	c, err := ocf.NewClientFromConfig(testCfg, nil)
	require.NoError(t, err)
	timeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	h := testDeviceHandler{}
	err = c.GetDevices(timeout, []string{}, &h)
	require.NoError(t, err)
}

func TestDeviceDiscoveryFilter(t *testing.T) {
	c, err := ocf.NewClientFromConfig(testCfg, nil)
	require.NoError(t, err)
	timeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	h := testDeviceHandler{}
	err = c.GetDevices(timeout, types.BaseTypes, &h)
	require.NoError(t, err)
}

type testDeviceHandler struct {
}

func (h *testDeviceHandler) Handle(ctx context.Context, client *device.Client) {
}

func (h *testDeviceHandler) Error(err error) {
}
