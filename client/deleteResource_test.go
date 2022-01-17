package client_test

import (
	"context"
	"testing"

	"github.com/plgd-dev/device/client"
	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/interfaces"
	"github.com/plgd-dev/device/test"
	"github.com/plgd-dev/device/test/resource/types"
	"github.com/stretchr/testify/require"
)

func createSwitch(ctx context.Context, t *testing.T, c *client.Client, deviceID string) {
	var got map[string]interface{}
	err := c.CreateResource(ctx, deviceID, test.TestResourceSwitchesHref, test.MakeSwitchResourceDefaultData(), &got)
	require.NoError(t, err)
	delete(got, "ins")
	require.Equal(t, test.MakeSwitchResourceData(map[string]interface{}{
		"href": test.TestResourceSwitchesInstanceHref("1"),
		"rep": map[interface{}]interface{}{
			"if":    []interface{}{interfaces.OC_IF_A, interfaces.OC_IF_BASELINE},
			"rt":    []interface{}{types.BINARY_SWITCH},
			"value": false,
		},
	}), got)
}

func TestClientDeleteResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.DevsimName)
	type args struct {
		deviceID string
		href     string
		opts     []client.DeleteOption
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				deviceID: deviceID,
				href:     test.TestResourceSwitchesInstanceHref("1"),
				opts:     []client.DeleteOption{client.WithDiscoveryConfiguration(core.DefaultDiscoveryConfiguration())},
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
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()
	deviceID, err = c.OwnDevice(ctx, deviceID, client.WithOTM(client.OTMType_JustWorks))
	require.NoError(t, err)
	defer disown(t, c, deviceID)

	createSwitch(ctx, t, c, deviceID)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.DeleteResource(ctx, tt.args.deviceID, tt.args.href, nil, tt.args.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
