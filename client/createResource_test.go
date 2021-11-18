package client_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/stretchr/testify/require"
)

func TestClientCreateResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimNetHost)
	type args struct {
		deviceID string
		href     string
		body     interface{}
		opts     []client.CreateOption
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
				href:     test.TestResourceSwitchesHref,
				body:     test.MakeSwitchResourceDefaultData(),
			},
			want: test.MakeSwitchResourceData(map[string]interface{}{
				"href": test.TestResourceSwitchesInstanceHref("1"),
				"rep": map[interface{}]interface{}{
					"if":    []interface{}{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
					"rt":    []interface{}{types.BINARY_SWITCH},
					"value": false,
				},
			}),
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
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	deviceID, err = c.OwnDevice(ctx, deviceID, client.WithOTM(client.OTMType_JustWorks))
	require.NoError(t, err)
	defer disown(t, c, deviceID)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]interface{}
			err := c.CreateResource(ctx, tt.args.deviceID, tt.args.href, tt.args.body, &got, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			delete(got, "ins")
			require.Equal(t, tt.want, got)
		})
	}
}
