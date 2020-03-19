package local

import (
	"context"

	codecOcf "github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	ocf "github.com/go-ocf/sdk/local/core"
)

func (c *Client) UpdateResourceWithCodec(
	ctx context.Context,
	deviceID string,
	href string,
	codec kitNetCoap.Codec,
	request interface{},
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	link, err := ocf.GetResourceLink(links, href)
	if err != nil {
		return err
	}

	return d.UpdateResourceWithCodec(ctx, link, codec, request, response, options...)
}

func (c *Client) UpdateResource(
	ctx context.Context,
	deviceID string,
	href string,
	request interface{},
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	var codec codecOcf.VNDOCFCBORCodec
	return c.UpdateResourceWithCodec(ctx, deviceID, href, codec, request, response, options...)
}
