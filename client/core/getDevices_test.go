package core_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	ocf "github.com/plgd-dev/device/client/core"

	"github.com/stretchr/testify/require"
)

func TestDeviceDiscovery(t *testing.T) {
	c := ocf.NewClient()
	timeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	h := testDeviceHandler{}
	err := c.GetDevicesV2(timeout, ocf.DefaultDiscoveryConfiguration(), &h)
	require.NoError(t, err)
}

type testDeviceHandler struct {
}

func (h *testDeviceHandler) Handle(ctx context.Context, d *ocf.Device) {
	defer d.Close(ctx)
}

func (h *testDeviceHandler) Error(err error) {
	fmt.Printf("testDeviceHandler.Error: %v\n", err)
}
