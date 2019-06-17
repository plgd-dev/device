package local

import (
	"context"

	"github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
)

func (c *Client) UpdateResource(
	ctx context.Context,
	deviceID, href string,
	request interface{},
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	codec := ocf.VNDOCFCBORCodec{}
	return c.UpdateResourceWithCodec(ctx, deviceID, href, codec, request, response, options...)
}

func (c *Client) UpdateResourceWithCodec(
	ctx context.Context,
	deviceID, href string,
	codec kitNetCoap.Codec,
	request interface{},
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	client, err := c.factory.NewClientFromCache()
	if err != nil {
		return err
	}
	options = append(options, kitNetCoap.WithAccept(codec.ContentFormat()))

	return client.Post(ctx, deviceID, href, codec, request, response, options...)
}
