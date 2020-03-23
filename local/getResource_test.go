package local_test

import (
	"context"
	"testing"
	"time"

	grpcTest "github.com/go-ocf/grpc-gateway/test"
	"github.com/go-ocf/sdk/local"
	"github.com/stretchr/testify/require"
)

func TestClient_GetResource(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(TestDeviceName)
	type args struct {
		deviceID string
		href     string
		opts     []local.GetOption
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
				href:     "/oc/con",
			},
			want: map[string]interface{}{
				"n": TestDeviceName,
			},
		},
		{
			name: "valid with interface",
			args: args{
				deviceID: deviceID,
				href:     "/oc/con",
				opts:     []local.GetOption{local.WithInterface("oic.if.baseline")},
			},
			wantErr: false,
			want: map[string]interface{}{
				"if": []interface{}{"oic.if.rw", "oic.if.baseline"},
				"n":  TestDeviceName,
				"rt": []interface{}{"oic.wk.con"},
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
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	c := NewTestClient()
	defer c.Close(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
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
