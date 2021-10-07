package core

import (
	"context"
	"fmt"

	kitNetCoap "github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/kit/v2/codec/ocf"
)

func (d *Device) DeleteResource(
	ctx context.Context,
	link schema.ResourceLink,
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	codec := ocf.VNDOCFCBORCodec{}
	return d.DeleteResourceWithCodec(ctx, link, codec, response, options...)
}

func (d *Device) DeleteResourceWithCodec(
	ctx context.Context,
	link schema.ResourceLink,
	codec kitNetCoap.Codec,
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	_, client, err := d.connectToEndpoints(ctx, link.GetEndpoints())
	if err != nil {
		return MakeInternal(fmt.Errorf("cannot delete resource %v: %w", link.Href, err))
	}
	options = append(options, kitNetCoap.WithAccept(codec.ContentFormat()))

	return client.DeleteResourceWithCodec(ctx, link.Href, codec, response, options...)
}
