package local_test

import (
	"context"
	"testing"
	"time"

	grpcTest "github.com/go-ocf/grpc-gateway/test"
	"github.com/go-ocf/sdk/local"
	"github.com/stretchr/testify/require"
)

func TestClient_UpdateResource(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(TestDeviceName)
	type args struct {
		deviceID string
		href     string
		data     interface{}
		opts     []local.UpdateOption
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
				href:     "/oc/con",
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
				href:     "/oc/con",
				data: map[string]interface{}{
					"n": t.Name() + "-valid with interface",
				},
				opts: []local.UpdateOption{local.WithInterface("oic.if.baseline")},
			},
			want: map[interface{}]interface{}{
				"n": t.Name() + "-valid with interface",
			},
		},
		{
			name: "valid - revert update",
			args: args{
				deviceID: deviceID,
				href:     "/oc/con",
				data: map[string]interface{}{
					"n": TestDeviceName,
				},
			},
			want: map[interface{}]interface{}{
				"n": TestDeviceName,
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
