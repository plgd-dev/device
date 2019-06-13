package local

import (
	"context"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/sdk/local/resource"
)

// coapContentFormat values can be found here
// https://github.com/go-ocf/go-coap/blob/a643abf9bcd9c4d033e63e7530e77d0f5f57dc54/message.go#L243
func (c *Client) GetResource(
	ctx context.Context,
	deviceID, href string,
	coapContentFormat uint16,
	options ...func(gocoap.Message),
) ([]byte, error) {
	var b []byte
	codec := coap.NoCodec{MediaType: coapContentFormat}
	err := c.getResource(ctx, deviceID, href, codec, &b, options...)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (c *Client) GetResourceVNDOCFCBOR(
	ctx context.Context,
	deviceID, href string,
	response interface{},
	options ...func(gocoap.Message),
) error {
	codec := coap.VNDOCFCBORCodec{}
	err := c.getResource(ctx, deviceID, href, codec, response, options...)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) getResource(
	ctx context.Context,
	deviceID, href string,
	codec resource.Codec,
	response interface{},
	options ...func(gocoap.Message),
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
