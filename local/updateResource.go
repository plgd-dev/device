package local

import (
	"context"

	codecOcf "github.com/go-ocf/kit/codec/ocf"
	"github.com/go-ocf/sdk/local/core"
)

func (c *Client) UpdateResource(
	ctx context.Context,
	deviceID string,
	href string,
	request interface{},
	response interface{},
	opts ...UpdateOption,
) error {
	cfg := updateOptions{
		json: codecOcf.VNDOCFCBORCodec{},
	}
	for _, o := range opts {
		cfg = o.applyOnUpdate(cfg)
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

	return d.UpdateResourceWithCodec(ctx, link, cfg.codec, request, response, cfg.opts...)
}
