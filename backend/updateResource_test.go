package backend_test

import (
	"context"
	"testing"
	"time"

	authTest "github.com/go-ocf/authorization/provider"
	grpcTest "github.com/go-ocf/grpc-gateway/test"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/backend"

	"github.com/stretchr/testify/require"
)

func TestClient_UpdateResource(t *testing.T) {
	deviceID := grpcTest.MustFindDeviceByName(grpcTest.TestDeviceName)
	type args struct {
		token    string
		deviceID string
		href     string
		data     interface{}
		opts     []backend.UpdateOption
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "valid - update value",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/oc/con",
				data: map[string]interface{}{
					"n": "devsim - valid update value",
				},
			},
			want: map[interface{}]interface{}{
				"n": "devsim - valid update value",
			},
		},
		{
			name: "valid - revert update",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/oc/con",
				data: map[string]interface{}{
					"n": grpcTest.TestDeviceName,
				},
			},
			want: map[interface{}]interface{}{
				"n": grpcTest.TestDeviceName,
			},
		},
		{
			name: "resourceInterface not supported",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/oc/con",
				opts:     []backend.UpdateOption{backend.WithInterface("oic.if.baseline")},
			},
			wantErr: true,
		},
		{
			name: "invalid href",
			args: args{
				token:    authTest.UserToken,
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
	ctx = kitNetGrpc.CtxWithToken(ctx, authTest.UserToken)

	tearDown := grpcTest.SetUp(ctx, t)
	defer tearDown()

	c := NewTestClient(t)
	defer c.Close(context.Background())
	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, grpcTest.GW_HOST, grpcTest.GetAllBackendResourceLinks())
	defer shutdownDevSim()

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
