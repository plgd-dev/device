package local

import (
	"context"

	"github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
)

// coapContentFormat values can be found here
// https://github.com/go-ocf/go-coap/blob/a643abf9bcd9c4d033e63e7530e77d0f5f57dc54/message.go#L243

func (c *Client) GetResource(
	ctx context.Context,
	deviceID, href string,
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	codec := ocf.VNDOCFCBORCodec{}
	return c.GetResourceWithCodec(ctx, deviceID, href, codec, response, options...)
}

func (c *Client) GetResourceWithCodec(
	ctx context.Context,
	deviceID, href string,
	codec kitNetCoap.Codec,
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	client, err := c.factory.NewClientFromCache()
	if err != nil {
		return err
	}
	options = append(options, kitNetCoap.WithAccept(codec.ContentFormat()))

	return client.Get(ctx, deviceID, href, codec, response, options...)
}
