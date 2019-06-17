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
	err := c.GetResourceWithCodec(ctx, deviceID, href, codec, response, options...)
	if err != nil {
		return err
	}
	return nil
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

	err = client.Get(ctx, deviceID, href, codec, response, options...)
	if err != nil {
		return err
	}

	return nil
}
