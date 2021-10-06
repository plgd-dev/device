package local

import (
	"context"

	codecOcf "github.com/plgd-dev/kit/v2/codec/ocf"
	"github.com/plgd-dev/sdk/v2/local/core"
)

func (c *Client) GetResource(
	ctx context.Context,
	deviceID string,
	href string,
	response interface{},
	opts ...GetOption,
) error {
	cfg := getOptions{
		codec: codecOcf.VNDOCFCBORCodec{},
	}
	for _, o := range opts {
		cfg = o.applyOnGet(cfg)
	}
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	link, err := core.GetResourceLink(links, href)
	if err != nil {
		return err
	}

	return d.GetResourceWithCodec(ctx, link, cfg.codec, response, cfg.opts...)
}
