package client

import (
	"context"

	"github.com/plgd-dev/device/client/core"
	kitNetCoap "github.com/plgd-dev/device/pkg/net/coap"
	codecOcf "github.com/plgd-dev/kit/v2/codec/ocf"
)

func (c *Client) CreateResource(
	ctx context.Context,
	deviceID string,
	href string,
	request interface{},
	response interface{},
	opts ...CreateOption,
) error {
	cfg := createOptions{
		codec: codecOcf.VNDOCFCBORCodec{},
		opts: []kitNetCoap.OptionFunc{
			kitNetCoap.WithInterface("oic.if.create"),
		},
	}
	for _, o := range opts {
		cfg = o.applyOnCreate(cfg)
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