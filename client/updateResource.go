package client

import (
	"context"

	"github.com/plgd-dev/device/v2/client/core"
	codecOcf "github.com/plgd-dev/device/v2/pkg/codec/ocf"
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
		codec:                  codecOcf.VNDOCFCBORCodec{},
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}

	for _, o := range opts {
		cfg = o.applyOnUpdate(cfg)
	}

	d, links, err := c.GetDeviceByMulticast(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}

	link, err := core.GetResourceLink(links, href)
	if err != nil {
		return err
	}

	return d.UpdateResourceWithCodec(ctx, link, cfg.codec, request, response, cfg.opts...)
}
