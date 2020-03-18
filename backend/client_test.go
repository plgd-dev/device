package backend_test

import (
	"context"
	"testing"

	"github.com/go-ocf/sdk/backend"
	"github.com/go-ocf/sdk/kiconnect/resource/types"
	"github.com/go-ocf/sdk/kiconnect/schema"
	"github.com/go-ocf/sdk/test"
	"github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/cbor"
	kit "github.com/go-ocf/kit/net/grpc"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/go-ocf/grpc-gateway/pb"
	grpcTest "github.com/go-ocf/grpc-gateway/test"
)

const (
	TestHref         = "test href"
	TestManufacturer = "Test Manufacturer"
)

var BackendTestCfg = backend.Config{
	GatewayAddress: grpcTest.GRPC_HOST,
	AccessTokenURL: grpcTest.AUTH_HOST,
}

func NewTestClient(t *testing.T) *backend.Client {
	app := testApplication{cas: grpcTest.GetRootCertificateAuthorities(t)}
	c, err := backend.NewClientFromConfig(&BackendTestCfg, &app)
	require.NoError(t, err)
	return c
}

func NewGateway(addr string) (*kit.Server, error) {
	s, err := kit.NewServer(addr)
	if err != nil {
		return nil, err
	}
	h := gatewayHandler{}

	pb.RegisterGrpcGatewayServer(s.Server, &h)

	return s, nil
}

type gatewayHandler struct {
}

func (h *gatewayHandler) GetDevices(req *pb.GetDevicesRequest, srv pb.GrpcGateway_GetDevicesServer) error {
	v := pb.Device{
		Id:               test.TestDeviceID,
		Name:             test.TestDeviceName,
		IsOnline:         true,
		ManufacturerName: []*pb.LocalizedString{&pb.LocalizedString{Value: TestManufacturer, Language: "en"}},
	}
	err := srv.Send(&v)
	if err != nil {
		return status.Errorf(status.Convert(err).Code(), "sending failed: %v", err)
	}
	return nil
}

func (h *gatewayHandler) GetResourceLinks(req *pb.GetResourceLinksRequest, srv pb.GrpcGateway_GetResourceLinksServer) error {
	err := srv.Send(&pb.ResourceLink{Href: "excluded", Types: []string{types.Device}, DeviceId: test.TestDeviceID})
	if err != nil {
		return status.Errorf(status.Convert(err).Code(), "sending failed: %v", err)
	}
	err = srv.Send(&pb.ResourceLink{Href: TestHref, Types: []string{"x.com.test.type"}, DeviceId: test.TestDeviceID})
	if err != nil {
		return status.Errorf(status.Convert(err).Code(), "sending failed: %v", err)
	}
	return nil
}

func (h *gatewayHandler) RetrieveResourcesValues(req *pb.RetrieveResourcesValuesRequest, srv pb.GrpcGateway_RetrieveResourcesValuesServer) error {
	err := sendResourceValue(srv, test.TestDeviceID, types.Device, schema.Device{
		SerialNumber: TestSerialNumber,
	})
	if err != nil {
		return err
	}
	err = sendResourceValue(srv, test.TestDeviceID, types.DataSource, schema.DataSource{
		ID: test.TestDataSourceID,
	})
	if err != nil {
		return err
	}
	return nil
}

func (h *gatewayHandler) UpdateResourcesValues(context.Context, *pb.UpdateResourceValuesRequest) (*pb.UpdateResourceValuesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func (h *gatewayHandler) SubscribeForEvents(pb.GrpcGateway_SubscribeForEventsServer) error {
	return status.Errorf(codes.Unimplemented, "not implemented")
}

func (h *gatewayHandler) RetrieveResourceFromDevice(context.Context, *pb.RetrieveResourceFromDeviceRequest) (*pb.RetrieveResourceFromDeviceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}

func sendResourceValue(srv pb.GrpcGateway_RetrieveResourcesValuesServer, deviceId, resourceType string, v interface{}) error {
	c, err := cbor.Encode(v)
	if err != nil {
		return status.Errorf(codes.Internal, "%v", err)
	}
	rv := pb.ResourceValue{
		ResourceId: &pb.ResourceId{DeviceId: deviceId},
		Types:      []string{resourceType},
		Content:    &pb.Content{ContentType: coap.AppCBOR.String(), Data: c},
	}
	err = srv.Send(&rv)
	if err != nil {
		return status.Errorf(status.Convert(err).Code(), "sending failed: %v", err)
	}
	return nil
}
