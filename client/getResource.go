package client

import (
	"context"

	"github.com/plgd-dev/device/client/core"
	codecOcf "github.com/plgd-dev/kit/v2/codec/ocf"
)

func (c *Client) GetResource(
	ctx context.Context,
	deviceID string,
	href string,
	response interface{},
	opts ...GetOption,
) error {
	cfg := getOptions{
		codec:                  codecOcf.VNDOCFCBORCodec{},
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnGet(cfg)
	}
	d, links, err := c.GetDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}

	link, err := core.GetResourceLink(links, href)
	if err != nil {
		return err
	}

	return d.GetResourceWithCodec(ctx, link, cfg.codec, response, cfg.opts...)
}
