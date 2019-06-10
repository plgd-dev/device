package local

import (
	"context"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/sdk/local/resource"
)

// coapContentFormat values can be found here
// https://github.com/go-ocf/go-coap/blob/a643abf9bcd9c4d033e63e7530e77d0f5f57dc54/message.go#L243
func (c *Client) UpdateResource(
	ctx context.Context,
	deviceID, href string,
	data []byte,
	coapContentFormat uint16,
	options ...func(gocoap.Message),
) ([]byte, error) {
	var b []byte
	codec := coap.NoCodec{MediaType: coapContentFormat}
	err := c.updateResource(ctx, deviceID, href, codec, data, &b, options...)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (c *Client) UpdateResourceCBOR(
	ctx context.Context,
	deviceID, href string,
	request interface{},
	response interface{},
	options ...func(gocoap.Message),
) error {
	codec := coap.CBORCodec{}
	err := c.updateResource(ctx, deviceID, href, codec, request, response, options...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) updateResource(
	ctx context.Context,
	deviceID, href string,
	codec resource.Codec,
	request interface{},
	response interface{},
	options ...func(gocoap.Message),
) error {
	client, err := c.factory.NewClientFromCache()
	if err != nil {
		return err
	}

	err = client.Post(ctx, deviceID, href, codec, request, response, options...)
	if err != nil {
		return err
	}

	return nil
}
