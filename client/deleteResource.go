package client

import (
	"context"

	"github.com/plgd-dev/device/client/core"
	codecOcf "github.com/plgd-dev/kit/v2/codec/ocf"
)

func (c *Client) DeleteResource(
	ctx context.Context,
	deviceID string,
	href string,
	response interface{},
	opts ...DeleteOption,
) error {
	cfg := deleteOptions{
		codec:                  codecOcf.VNDOCFCBORCodec{},
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnDelete(cfg)
	}

	d, links, err := c.GetRefDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	link, err := core.GetResourceLink(links, href)
	if err != nil {
		return err
	}

	return d.DeleteResourceWithCodec(ctx, link, cfg.codec, response, cfg.opts...)
}
