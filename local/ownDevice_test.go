package local_test

import (
	"context"
	"testing"
	"time"

	ocf "github.com/go-ocf/sdk/local"
	"github.com/go-ocf/sdk/schema"
	"github.com/stretchr/testify/require"
)

//docker rm -f devsim; docker run -d -t --network=host --name devsim dockerhub.kistler.com/kiconnect/kiconnect-device-simulator:2.2.7-secure-dbg --name devsim --di '00000000-cafe-baba-0000-000000000000'

func TestClient_ownDevice(t *testing.T) {
	type args struct {
		deviceID string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				deviceID: "00000000-cafe-baba-0000-000000000000",
			},
		},
	}

	c, err := ocf.NewClientFromConfig(testCfg, nil)
	require := require.New(t)
	require.NoError(err)

	/*
		timeout, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		h := testOnboardDeviceHandler{}
		err = c.GetDevices(timeout, []string{"oic.d.cloudDevice"}, &h)
		require.NoError(err)
		deviceIds := h.PopDeviceIds()
		require.NotEmpty(deviceIds)
	*/

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := c.OwnDevice(timeout, tt.args.deviceID, schema.JustWorks, 10*time.Second)
			cancel()
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}
