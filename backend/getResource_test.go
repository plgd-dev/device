package backend_test

import (
	"context"
	"testing"
	"time"

	kitNetGrpc "github.com/go-ocf/kit/net/grpc"

	authTest "github.com/go-ocf/authorization/provider"
	grpcTest "github.com/go-ocf/grpc-gateway/test"
	"github.com/go-ocf/sdk/backend"
	"github.com/go-ocf/sdk/test"
	"github.com/stretchr/testify/require"
)

func TestClient_GetResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		token    string
		deviceID string
		href     string
		opts     []backend.ResourceOption
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
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/light/1",
			},
			want: map[interface{}]interface{}{
				"fr":       false,
				"identify": false,
				"if": []interface{}{
					"oic.if.r", "oic.if.rw", "oic.if.baseline",
				},
				"rb": false,
				"rt": []interface{}{
					"oic.wk.mnt", "x.com.kistler.kiconnect.identify",
				},
			},
		},
		{
			name: "valid with skip shadow",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/light/1",
				opts:     []backend.ResourceOption{backend.WithSkipShadow()},
			},
			want: map[interface{}]interface{}{
				"fr":       false,
				"identify": false,
				"if": []interface{}{
					"oic.if.r", "oic.if.rw", "oic.if.baseline",
				},
				"rb": false,
				"rt": []interface{}{
					"oic.wk.mnt", "x.com.kistler.kiconnect.identify",
				},
			},
		},
		{
			name: "valid with interface",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/light/1",
				opts:     []backend.ResourceOption{backend.WithInterface("oic.if.rw")},
			},
			wantErr: false,
			want: map[interface{}]interface{}{
				"fr":       false,
				"identify": false,
				"rb":       false,
			},
		},
		{
			name: "valid with interface and skip shadow",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/light/1",
				opts:     []backend.ResourceOption{backend.WithSkipShadow(), backend.WithInterface("oic.if.rw")},
			},
			wantErr: false,
			want: map[interface{}]interface{}{
				"fr":       false,
				"identify": false,
				"rb":       false,
			},
		},
		{
			name: "invalid href",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
				href:     "/invalid/href",
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

	shutdownDevSim := grpcTest.OnboardDevSim(ctx, t, c.GrpcGatewayClient(), deviceID, grpcTest.GW_HOST)
	defer shutdownDevSim()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			defer cancel()
			var got interface{}
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
