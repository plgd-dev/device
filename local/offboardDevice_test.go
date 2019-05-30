package local_test

import (
	"context"
	"testing"
	"time"

	ocf "github.com/go-ocf/sdk/local"
	"github.com/stretchr/testify/require"
)

func TestClient_OffboardDevice(t *testing.T) {
	c, err := ocf.NewClientFromConfig(testCfg, nil)
	require := require.New(t)
	require.NoError(err)

	timeout, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	h := testOnboardDeviceHandler{}
	err = c.GetDevices(timeout, []string{"oic.d.cloudDevice"}, &h)
	require.NoError(err)
	deviceIds := h.PopDeviceIds()
	require.NotEmpty(deviceIds)

	func() {
		for deviceId, _ := range deviceIds {
			timeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			err = c.OffboardDevice(timeout, deviceId)
			require.NoError(err)
		}
	}()

}
