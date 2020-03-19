package local

import (
	"context"

	codecOcf "github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	ocf "github.com/go-ocf/sdk/local/core"
)

func (c *Client) GetResourceWithCodec(
	ctx context.Context,
	deviceID string,
	href string,
	codec kitNetCoap.Codec,
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

	return d.GetResourceWithCodec(ctx, link, codec, response, options...)
}

func (c *Client) GetResource(
	ctx context.Context,
	deviceID string,
	href string,
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	var codec codecOcf.VNDOCFCBORCodec
	return c.GetResourceWithCodec(ctx, deviceID, href, codec, response, options...)
}
