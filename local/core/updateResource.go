package core

import (
	"context"
	"fmt"

	"github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/schema"
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
	client, err := d.connectToEndpoints(ctx, link.GetEndpoints())
	if err != nil {
		return fmt.Errorf("cannot update resource %v: %w", link.Href, err)
	}
	options = append(options, kitNetCoap.WithAccept(codec.ContentFormat()))

	return client.UpdateResourceWithCodec(ctx, link.Href, codec, request, response, options...)
}
