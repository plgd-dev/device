package cloud

import (
	"context"

	"github.com/go-ocf/grpc-gateway/pb"
	"github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/strings"
)

// NewClient constructs a new OCF cloud client.
func NewClient(gateway pb.GrpcGatewayClient) *Client {
	return &Client{gateway: gateway}
}

// Client for communication with the OCF Cloud.
type Client struct {
	gateway pb.GrpcGatewayClient
}

func (c *Client) GetDevicesViaCallback(ctx context.Context, token string, deviceIDs, resourceTypes []string, callback func(pb.Device)) error {
	it := c.GetDevices(ctx, token, deviceIDs, resourceTypes...)
	defer it.Close()
	var v pb.Device
	for it.Next(&v) {
		callback(v)
	}
	return it.Err
}

func (c *Client) GetResourceLinksViaCallback(ctx context.Context, token string, deviceIDs, resourceTypes []string, callback func(pb.ResourceLink)) error {
	it := c.GetResourceLinks(ctx, token, deviceIDs, resourceTypes...)
	defer it.Close()
	var v pb.ResourceLink
	for it.Next(&v) {
		callback(v)
	}
	return it.Err
}

type TypeCallback struct {
	Type     string
	Callback func(pb.ResourceValue)
}

func MakeTypeCallback(resourceType string, callback func(pb.ResourceValue)) TypeCallback {
	return TypeCallback{Type: resourceType, Callback: callback}
}

func (c *Client) RetrieveResourcesByType(
	ctx context.Context,
	token string,
	deviceIDs []string,
	typeCallbacks ...TypeCallback,
) error {
	tc := make(map[string]func(pb.ResourceValue), len(typeCallbacks))
	resourceTypes := make([]string, 0, len(typeCallbacks))
	for _, c := range typeCallbacks {
		tc[c.Type] = c.Callback
		resourceTypes = append(resourceTypes, c.Type)
	}

	it := c.RetrieveResources(ctx, token, deviceIDs, resourceTypes...)
	defer it.Close()
	var v pb.ResourceValue
	for it.Next(&v) {
		for _, rt := range resourceTypes {
			if strings.SliceContains(v.Types, rt) {
				tc[rt](v)
				break
			}
		}
	}
	return it.Err
}

func (c *Client) GetDevices(ctx context.Context, token string, deviceIDs []string, resourceTypes ...string) *grpc.Iterator {
	auth := pb.AuthorizationContext{AccessToken: token}
	ctx = grpc.CtxWithToken(ctx, token)
	r := pb.GetDevicesRequest{DeviceIdsFilter: deviceIDs, TypeFilter: resourceTypes, AuthorizationContext: &auth}
	return grpc.NewIterator(c.gateway.GetDevices(ctx, &r))
}

func (c *Client) GetResourceLinks(ctx context.Context, token string, deviceIDs []string, resourceTypes ...string) *grpc.Iterator {
	auth := pb.AuthorizationContext{AccessToken: token}
	ctx = grpc.CtxWithToken(ctx, token)
	r := pb.GetResourceLinksRequest{DeviceIdsFilter: deviceIDs, TypeFilter: resourceTypes, AuthorizationContext: &auth}
	return grpc.NewIterator(c.gateway.GetResourceLinks(ctx, &r))
}

func (c *Client) RetrieveResources(ctx context.Context, token string, deviceIDs []string, resourceTypes ...string) *grpc.Iterator {
	auth := pb.AuthorizationContext{AccessToken: token}
	ctx = grpc.CtxWithToken(ctx, token)
	r := pb.RetrieveResourcesValuesRequest{DeviceIdsFilter: deviceIDs, TypeFilter: resourceTypes, AuthorizationContext: &auth}
	return grpc.NewIterator(c.gateway.RetrieveResourcesValues(ctx, &r))
}
