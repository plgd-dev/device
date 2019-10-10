package cloud

import (
	"context"

	"github.com/go-ocf/resource-aggregate/cqrs"

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

// TypeCallback calls callback for type, if callback returns false iterator over resource values will be terminated.
type TypeCallback struct {
	Type     string
	Callback func(pb.ResourceValue) bool
}

func MakeTypeCallback(resourceType string, callback func(pb.ResourceValue) bool) TypeCallback {
	return TypeCallback{Type: resourceType, Callback: callback}
}

func (c *Client) RetrieveResourcesByType(
	ctx context.Context,
	token string,
	deviceIDs []string,
	typeCallbacks ...TypeCallback,
) error {
	tc := make(map[string]func(pb.ResourceValue) bool, len(typeCallbacks))
	resourceTypes := make([]string, 0, len(typeCallbacks))
	for _, c := range typeCallbacks {
		tc[c.Type] = c.Callback
		resourceTypes = append(resourceTypes, c.Type)
	}

	it := c.RetrieveResources(ctx, token, nil, deviceIDs, resourceTypes...)
	defer it.Close()
	var v pb.ResourceValue
	for it.Next(&v) {
		for _, rt := range resourceTypes {
			if strings.SliceContains(v.Types, rt) {
				if !tc[rt](v) {
					return nil
				}
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

func (c *Client) RetrieveResources(ctx context.Context, token string, resourceIDs []*pb.ResourceId, deviceIDs []string, resourceTypes ...string) *grpc.Iterator {
	auth := pb.AuthorizationContext{AccessToken: token}
	ctx = grpc.CtxWithToken(ctx, token)
	r := pb.RetrieveResourcesValuesRequest{ResourceIdsFilter: resourceIDs, DeviceIdsFilter: deviceIDs, TypeFilter: resourceTypes, AuthorizationContext: &auth}
	return grpc.NewIterator(c.gateway.RetrieveResourcesValues(ctx, &r))
}

type ResourceIDCallback struct {
	ResourceID *pb.ResourceId
	Callback   func(pb.ResourceValue)
}

func MakeResourceIDCallback(deviceID, href string, callback func(pb.ResourceValue)) ResourceIDCallback {
	return ResourceIDCallback{ResourceID: &pb.ResourceId{
		DeviceId:         deviceID,
		ResourceLinkHref: href,
	}, Callback: callback}
}

func (c *Client) RetrieveResourcesByResourceIDs(
	ctx context.Context,
	token string,
	resourceIDsCallbacks ...ResourceIDCallback,
) error {
	tc := make(map[string]func(pb.ResourceValue), len(resourceIDsCallbacks))
	resourceIDs := make([]*pb.ResourceId, 0, len(resourceIDsCallbacks))
	for _, c := range resourceIDsCallbacks {
		tc[cqrs.MakeResourceId(c.ResourceID.DeviceId, c.ResourceID.ResourceLinkHref)] = c.Callback
		resourceIDs = append(resourceIDs, c.ResourceID)
	}

	it := c.RetrieveResources(ctx, token, resourceIDs, nil)
	defer it.Close()
	var v pb.ResourceValue
	for it.Next(&v) {
		c, ok := tc[cqrs.MakeResourceId(v.GetResourceId().GetDeviceId(), v.GetResourceId().GetResourceLinkHref())]
		if ok {
			c(v)
		}
	}
	return it.Err
}

func (c *Client) UpdateResource(
	ctx context.Context,
	token string,
	resourceID pb.ResourceId,
	content pb.Content,
) (*pb.UpdateResourceValuesResponse, error) {
	auth := pb.AuthorizationContext{AccessToken: token}
	ctx = grpc.CtxWithToken(ctx, token)
	r := pb.UpdateResourceValuesRequest{
		ResourceId:           &resourceID,
		Content:              &content,
		AuthorizationContext: &auth,
	}
	return c.gateway.UpdateResourcesValues(ctx, &r)
}
