package core

import (
	"context"
	"fmt"

	kitNetCoap "github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/kit/v2/codec/ocf"
)

func (d *Device) UpdateResource(
	ctx context.Context,
	link schema.ResourceLink,
	request interface{},
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	codec := ocf.VNDOCFCBORCodec{}
	return d.UpdateResourceWithCodec(ctx, link, codec, request, response, options...)
}

func (d *Device) UpdateResourceWithCodec(
	ctx context.Context,
	link schema.ResourceLink,
	codec kitNetCoap.Codec,
	request interface{},
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	_, client, err := d.connectToEndpoints(ctx, link.GetEndpoints())
	if err != nil {
		return MakeInternal(fmt.Errorf("cannot update resource %v: %w", link.Href, err))
	}
	options = append(options, kitNetCoap.WithAccept(codec.ContentFormat()))

	return client.UpdateResourceWithCodec(ctx, link.Href, codec, request, response, options...)
}
