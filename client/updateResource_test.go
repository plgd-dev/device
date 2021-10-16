package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/schema/configuration"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/require"
)

func TestClientUpdateResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		deviceID string
		href     string
		data     interface{}
		opts     []client.UpdateOption
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				data: map[string]interface{}{
					"n": t.Name() + "-valid",
				},
			},
			want: map[interface{}]interface{}{
				"n": t.Name() + "-valid",
			},
		},
		{
			name: "valid with interface",
			args: args{
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				data: map[string]interface{}{
					"n": t.Name() + "-valid with interface",
				},
				opts: []client.UpdateOption{client.WithInterface(interfaces.OC_IF_BASELINE)},
			},
			want: map[interface{}]interface{}{
				"n": t.Name() + "-valid with interface",
			},
		},
		{
			name: "valid - revert update",
			args: args{
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				data: map[string]interface{}{
					"n": test.TestDeviceName,
				},
			},
			want: map[interface{}]interface{}{
				"n": test.TestDeviceName,
			},
		},
		{
			name: "invalid href",
			args: args{
				deviceID: deviceID,
				href:     "/invalid/href",
				data: map[string]interface{}{
					"n": "devsim",
				},
			},
			wantErr: true,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	c := NewTestClient()
	defer c.Close(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			var got interface{}
			err := c.UpdateResource(ctx, tt.args.deviceID, tt.args.href, tt.args.data, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
