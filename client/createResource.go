package client

import (
	"context"

	"github.com/plgd-dev/device/client/core"
	codecOcf "github.com/plgd-dev/device/pkg/codec/ocf"
	"github.com/plgd-dev/device/pkg/net/coap"
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
		opts: []coap.OptionFunc{
			coap.WithInterface("oic.if.create"),
		},
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnCreate(cfg)
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

	return d.UpdateResourceWithCodec(ctx, link, cfg.codec, request, response, cfg.opts...)
}
