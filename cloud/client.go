package cloud

import (
	"context"
	"fmt"

	"github.com/go-ocf/resource-aggregate/cqrs"

	"github.com/go-ocf/grpc-gateway/pb"
	codecOcf "github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/kit/net/grpc"
	"github.com/go-ocf/kit/strings"
)

// NewClient constructs a new OCF cloud client. For every call there is expected jwt token for grpc stored in context.
func NewClient(gateway pb.GrpcGatewayClient) *Client {
	return &Client{gateway: gateway}
}

// Client for communication with the OCF Cloud.
type Client struct {
	gateway pb.GrpcGatewayClient
}

// GetDevicesViaCallback returns devices. JWT token must be stored in context for grpc call.
func (c *Client) GetDevicesViaCallback(ctx context.Context, deviceIDs, resourceTypes []string, callback func(pb.Device)) error {
	it := c.GetDevices(ctx, deviceIDs, resourceTypes...)
	defer it.Close()
	var v pb.Device
	for it.Next(&v) {
		callback(v)
	}
	return it.Err
}

// GetResourceLinksViaCallback returns resource links of devices. JWT token must be stored in context for grpc call.
func (c *Client) GetResourceLinksViaCallback(ctx context.Context, deviceIDs, resourceTypes []string, callback func(pb.ResourceLink)) error {
	it := c.GetResourceLinks(ctx, deviceIDs, resourceTypes...)
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

// RetrieveResourcesByType gets contents of resources by resource types. JWT token must be stored in context for grpc call.
func (c *Client) RetrieveResourcesByType(
	ctx context.Context,
	deviceIDs []string,
	typeCallbacks ...TypeCallback,
) error {
	tc := make(map[string]func(pb.ResourceValue), len(typeCallbacks))
	resourceTypes := make([]string, 0, len(typeCallbacks))
	for _, c := range typeCallbacks {
		tc[c.Type] = c.Callback
		resourceTypes = append(resourceTypes, c.Type)
	}

	it := c.RetrieveResources(ctx, nil, deviceIDs, resourceTypes...)
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

// GetDevices gets devices. JWT token must be stored in context for grpc call.
func (c *Client) GetDevices(ctx context.Context, deviceIDs []string, resourceTypes ...string) *grpc.Iterator {
	r := pb.GetDevicesRequest{DeviceIdsFilter: deviceIDs, TypeFilter: resourceTypes}
	return grpc.NewIterator(c.gateway.GetDevices(ctx, &r))
}

// GetResourceLinks gets devices. JWT token must be stored in context for grpc call.
func (c *Client) GetResourceLinks(ctx context.Context, deviceIDs []string, resourceTypes ...string) *grpc.Iterator {
	r := pb.GetResourceLinksRequest{DeviceIdsFilter: deviceIDs, TypeFilter: resourceTypes}
	return grpc.NewIterator(c.gateway.GetResourceLinks(ctx, &r))
}

// RetrieveResources gets resources contents. JWT token must be stored in context for grpc call.
func (c *Client) RetrieveResources(ctx context.Context, resourceIDs []*pb.ResourceId, deviceIDs []string, resourceTypes ...string) *grpc.Iterator {
	r := pb.RetrieveResourcesValuesRequest{ResourceIdsFilter: resourceIDs, DeviceIdsFilter: deviceIDs, TypeFilter: resourceTypes}
	return grpc.NewIterator(c.gateway.RetrieveResourcesValues(ctx, &r))
}

type ResourceIDCallback struct {
	ResourceID *pb.ResourceId
	Callback   func(pb.ResourceValue)
}

func MakeResourceIDCallback(deviceID, href string, callback func(pb.ResourceValue)) ResourceIDCallback {
	return ResourceIDCallback{ResourceID: &pb.ResourceId{
		DeviceID:         deviceID,
		ResourceLinkHref: href,
	}, Callback: callback}
}

// RetrieveResourcesByResourceIDs gets resources contents by resourceIDs. JWT token must be stored in context for grpc call.
func (c *Client) RetrieveResourcesByResourceIDs(
	ctx context.Context,
	resourceIDsCallbacks ...ResourceIDCallback,
) error {
	tc := make(map[string]func(pb.ResourceValue), len(resourceIDsCallbacks))
	resourceIDs := make([]*pb.ResourceId, 0, len(resourceIDsCallbacks))
	for _, c := range resourceIDsCallbacks {
		tc[cqrs.MakeResourceId(c.ResourceID.DeviceID, c.ResourceID.ResourceLinkHref)] = c.Callback
		resourceIDs = append(resourceIDs, c.ResourceID)
	}

	it := c.RetrieveResources(ctx, resourceIDs, nil)
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

// UpdateResourceWithContent updates resource with content. JWT token must be stored in context for grpc call.
func (c *Client) UpdateResourceWithContent(
	ctx context.Context,
	resourceID pb.ResourceId,
	content pb.Content,
) (*pb.UpdateResourceValuesResponse, error) {
	r := pb.UpdateResourceValuesRequest{
		ResourceId: &resourceID,
		Content:    &content,
	}
	return c.gateway.UpdateResourcesValues(ctx, &r)
}

// UpdateResourceWithCodec update resource with codec. JWT token must be stored in context for grpc call.
func (c *Client) UpdateResourceWithCodec(
	ctx context.Context,
	resourceID pb.ResourceId,
	interfaceFilter string,
	codec kitNetCoap.Codec,
	request interface{},
	response interface{},
) error {
	if interfaceFilter != "" {
		return fmt.Errorf("interface is not supported")
	}
	data, err := codec.Encode(request)
	if err != nil {
		return err
	}

	resp, err := c.UpdateResourceWithContent(ctx, resourceID, pb.Content{
		Data:        data,
		ContentType: codec.ContentFormat().String(),
	})
	if err != nil {
		return fmt.Errorf("cannot update resource %+v: %w", resourceID, err)
	}

	return DecodeContentWithCodec(codec, resp.GetContent().GetContentType(), resp.GetContent().GetData(), response)
}

// UpdateResource updates content vic OCF-CBOR format. JWT token must be stored in context for grpc call.
func (c *Client) UpdateResource(
	ctx context.Context,
	resourceID pb.ResourceId,
	interfaceFilter string,
	request interface{},
	response interface{},
) error {
	var codec codecOcf.VNDOCFCBORCodec
	return c.UpdateResourceWithCodec(ctx, resourceID, interfaceFilter, codec, request, response)
}
