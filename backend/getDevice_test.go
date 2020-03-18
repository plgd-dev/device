package backend_test

import (
	"context"
	"crypto/x509"
	"sort"
	"testing"
	"time"

	kitNetGrpc "github.com/go-ocf/kit/net/grpc"

	authTest "github.com/go-ocf/authorization/provider"
	"github.com/go-ocf/grpc-gateway/pb"
	grpcTest "github.com/go-ocf/grpc-gateway/test"
	"github.com/go-ocf/sdk/backend"
	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/test"
	"github.com/stretchr/testify/require"
)

const TestTimeout = time.Second * 8
const DeviceSimulatorIdNotFound = "00000000-0000-0000-0000-000000000111"

type sortResourcesByHref []pb.ResourceLink

func (a sortResourcesByHref) Len() int      { return len(a) }
func (a sortResourcesByHref) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortResourcesByHref) Less(i, j int) bool {
	return a[i].Href < a[j].Href
}

func sortResources(s []pb.ResourceLink) []pb.ResourceLink {
	v := sortResourcesByHref(s)
	sort.Sort(v)
	return v
}

type testApplication struct {
	cas []*x509.Certificate
}

func (a *testApplication) GetRootCertificateAuthorities() ([]*x509.Certificate, error) {
	return a.cas, nil
}

func convertSchemaToPb(deviceID string, s []schema.ResourceLink) []pb.ResourceLink {
	r := make([]pb.ResourceLink, 0, len(s))
	for _, l := range s {
		r = append(r, pb.ResourceLink{
			Href:       l.Href,
			DeviceId:   deviceID,
			Types:      l.ResourceTypes,
			Interfaces: l.Interfaces,
		})
	}
	return r
}

func NewTestDeviceSimulator(deviceID, deviceName string) backend.DeviceDetails {
	return backend.DeviceDetails{
		ID: deviceID,
		Device: pb.Device{
			Id:   deviceID,
			Name: deviceName,
		},
		Resources: sortResources(convertSchemaToPb(deviceID, test.TestDevsimResources)),
	}
}

func TestClient_GetDevice(t *testing.T) {
	deviceID := test.MustFindDeviceByName(test.TestDeviceName)
	type args struct {
		token    string
		deviceID string
	}
	tests := []struct {
		name    string
		args    args
		want    backend.DeviceDetails
		wantErr bool
	}{
		{
			name: "valid",
			args: args{
				token:    authTest.UserToken,
				deviceID: deviceID,
			},
			want: NewTestDeviceSimulator(deviceID, test.TestDeviceName),
		},
		{
			name: "not-found",
			args: args{
				token:    authTest.UserToken,
				deviceID: "not-found",
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
			got, err := c.GetDevice(ctx, tt.args.deviceID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got.Resources = sortResources(got.Resources)
			require.Equal(t, tt.want, got)
		})
	}
}
