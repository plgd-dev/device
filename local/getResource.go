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
	interfaceFilter string,
	coapContentFormat uint16,
) ([]byte, error) {
	var b []byte
	codec := coap.NoCodec{MediaType: coapContentFormat}
	err := c.getResource(ctx, deviceID, href, interfaceFilter, codec, &b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (c *Client) GetResourceCBOR(
	ctx context.Context,
	deviceID, href string,
	interfaceFilter string,
	response interface{},
) error {
	codec := coap.CBORCodec{}
	err := c.getResource(ctx, deviceID, href, interfaceFilter, codec, response)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) getResource(
	ctx context.Context,
	deviceID, href string,
	interfaceFilter string,
	codec resource.Codec,
	response interface{},
) error {
	var options []func(gocoap.Message)
	if interfaceFilter != "" {
		options = append(options, func(req gocoap.Message) {
			req.AddOption(gocoap.URIQuery, "if="+interfaceFilter)
		})
	}

	client, err := c.factory.NewClientFromCache(codec)
	if err != nil {
		return err
	}

	err = client.Get(ctx, deviceID, href, response, options...)
	if err != nil {
		return err
	}

	return nil
}
