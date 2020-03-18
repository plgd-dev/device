package backend_test

import (
	"context"
	"testing"
	"time"

	authTest "github.com/go-ocf/authorization/provider"
	grpcTest "github.com/go-ocf/grpc-gateway/test"
	kitNetGrpc "github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/sdk/backend"
	"github.com/go-ocf/sdk/test"
	"github.com/stretchr/testify/require"
)

func sortDevices(s map[string]backend.DeviceDetails) map[string]backend.DeviceDetails {
	for key, x := range s {
		x.Resources = sortResources(x.Resources)
		s[key] = x
	}

	return s
}

func TestClient_GetDevices(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		token      string
		deviceIDs  []string
		typeFilter []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]backend.DeviceDetails
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				token: authTest.UserToken,
			},
			want: map[string]backend.DeviceDetails{
				deviceID: NewTestDeviceSimulator(deviceID, test.TestDeviceName),
			},
		},
		{
			name: "not-found - OK",
			args: args{
				token:      authTest.UserToken,
				typeFilter: []string{"not-found"},
			},
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
			got, err := c.GetDevices(ctx, tt.args.deviceIDs, tt.args.typeFilter)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got = sortDevices(got)
			require.Equal(t, tt.want, got)
		})
	}
}
