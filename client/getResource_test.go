package client_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/schema/configuration"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/require"
)

func TestClientGetResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimNetHost)
	type args struct {
		deviceID string
		href     string
		opts     []client.GetOption
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceID: deviceID,
				href:     configuration.ResourceURI,
			},
			want: map[string]interface{}{
				"n": test.DevsimNetHost,
			},
		},
		{
			name: "valid with interface",
			args: args{
				deviceID: deviceID,
				href:     configuration.ResourceURI,
				opts:     []client.GetOption{client.WithInterface(interfaces.OC_IF_BASELINE)},
			},
			wantErr: false,
			want: map[string]interface{}{
				"if": []interface{}{interfaces.OC_IF_RW, interfaces.OC_IF_BASELINE},
				"n":  test.DevsimNetHost,
				"rt": []interface{}{configuration.ResourceType},
			},
		},
		{
			name: "invalid href",
			args: args{
				deviceID: deviceID,
				href:     "/invalid/href",
			},
			wantErr: true,
		},
		{
			name: "invalid deviceID",
			args: args{
				deviceID: "notfound",
				href:     device.ResourceURI,
			},
			wantErr: true,
		},
	}

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		err := c.Close(context.Background())
		require.NoError(t, err)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout*8)
	defer cancel()
	deviceID, err = c.OwnDevice(ctx, deviceID)
	require.NoError(t, err)
	defer disown(t, c, deviceID)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]interface{}
			err := c.GetResource(ctx, tt.args.deviceID, tt.args.href, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
