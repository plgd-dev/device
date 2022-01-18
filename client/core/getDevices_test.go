package core_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/plgd-dev/device/client/core"
	"github.com/stretchr/testify/require"
)

func TestDeviceDiscovery(t *testing.T) {
	c := core.NewClient()
	timeout, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	h := testDeviceHandler{}
	err := c.GetDevicesV2(timeout, core.DefaultDiscoveryConfiguration(), &h)
	require.NoError(t, err)
}

type testDeviceHandler struct {
}

func (h *testDeviceHandler) Handle(ctx context.Context, d *core.Device) {
	defer func() {
		if errClose := d.Close(ctx); errClose != nil {
			h.Error(fmt.Errorf("testDeviceHandler.Handle: %w", errClose))
		}
	}()
}

func (h *testDeviceHandler) Error(err error) {
	fmt.Printf("testDeviceHandler.Error: %v\n", err)
}
