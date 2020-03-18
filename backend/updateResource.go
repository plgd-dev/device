package backend

import (
	"context"
	"fmt"

	"github.com/go-ocf/grpc-gateway/pb"
	codecOcf "github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
)

// UpdateResourceWithCodec update resource with codec.
func (c *Client) UpdateResourceWithCodec(
	ctx context.Context,
	deviceID string,
	href string,
	codec kitNetCoap.Codec,
	request interface{},
	response interface{},
) error {
	data, err := codec.Encode(request)
	if err != nil {
		return err
	}
	r := pb.UpdateResourceValuesRequest{
		ResourceId: &pb.ResourceId{
			DeviceId:         deviceID,
			ResourceLinkHref: href,
		},
		Content: &pb.Content{
			Data:        data,
			ContentType: codec.ContentFormat().String(),
		},
	}

	resp, err := c.gateway.UpdateResourcesValues(ctx, &r)
	if err != nil {
		return fmt.Errorf("cannot update resource /%v/%v: %w", deviceID, href, err)
	}

	return DecodeContentWithCodec(codec, resp.GetContent().GetContentType(), resp.GetContent().GetData(), response)
}

// UpdateResource updates content in OCF-CBOR format.
func (c *Client) UpdateResource(
	ctx context.Context,
	deviceID string,
	href string,
	request interface{},
	response interface{},
) error {
	var codec codecOcf.VNDOCFCBORCodec
	return c.UpdateResourceWithCodec(ctx, deviceID, href, codec, request, response)
}
