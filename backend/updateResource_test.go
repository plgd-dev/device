package backend_test

import (
	"context"
	"testing"
	"time"

	authTest "github.com/go-ocf/authorization/provider"
	"github.com/go-ocf/go-coap"
	grpcTest "github.com/go-ocf/grpc-gateway/test"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/backend"
	"github.com/go-ocf/sdk/test"

	"github.com/stretchr/testify/require"
)

func TestClient_UpdateResource(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		token             string
		deviceID          string
		href              string
		resourceInterface string
		data              []byte
		coapContentFormat uint16
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
				token: authTest.UserToken,
				href:  "/" + TestDeviceSimulator.GetId() + "/kic/con",
				data: test.EncodeToCbor(t, map[string]interface{}{
					"n": "devsim - valid update value",
				}),
				coapContentFormat: uint16(coap.AppCBOR),
			},
			want: map[interface{}]interface{}{
				"if": []interface{}{
					"oic.if.r", "oic.if.rw", "oic.if.baseline",
				},
				"n": "devsim - valid update value",
				"rt": []interface{}{
					"oic.wk.con",
				},
			},
		},
		{
			name: "valid - revert update",
			args: args{
				token: authTest.UserToken,
				href:  "/" + TestDeviceSimulator.GetId() + "/kic/con",
				data: test.EncodeToCbor(t, map[string]interface{}{
					"n": test.TestDeviceName,
				}),
				coapContentFormat: uint16(coap.AppCBOR),
			},
			want: map[interface{}]interface{}{
				"if": []interface{}{
					"oic.if.r", "oic.if.rw", "oic.if.baseline",
				},
				"n": test.TestDeviceName,
				"rt": []interface{}{
					"oic.wk.con",
				},
			},
		},
		{
			name: "resourceInterface not supported",
			args: args{
				token:             authTest.UserToken,
				href:              "/" + TestDeviceSimulator.GetId() + "/kic/con",
				coapContentFormat: uint16(coap.AppCBOR),
				resourceInterface: "oic.if.r",
			},
			wantErr: true,
		},
		{
			name: "invalid href",
			args: args{
				token:             authTest.UserToken,
				href:              "/" + TestDeviceSimulator.GetId() + "/invalid/href",
				coapContentFormat: uint16(coap.AppCBOR),
				data: test.EncodeToCbor(t, map[string]interface{}{
					"n": "devsim",
				}),
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
			got, err := c.UpdateResource(ctx, tt.args.href, tt.args.data, tt.args.coapContentFormat, backend.WithInterface(tt.args.resourceInterface))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, test.DecodeCbor(t, got))
		})
	}
}
