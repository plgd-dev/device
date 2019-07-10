package local

import (
	"context"

	"github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
)

func (d *Device) UpdateResource(
	ctx context.Context,
	href string,
	request interface{},
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	codec := ocf.VNDOCFCBORCodec{}
	return d.UpdateResourceWithCodec(ctx, href, codec, request, response, options...)
}

func (d *Device) UpdateResourceWithCodec(
	ctx context.Context,
	href string,
	codec kitNetCoap.Codec,
	request interface{},
	response interface{},
	options ...kitNetCoap.OptionFunc,
) error {
	client, err := d.connect(ctx, href)
	if err != nil {
		return err
	}
	options = append(options, kitNetCoap.WithAccept(codec.ContentFormat()))

	return client.UpdateResourceWithCodec(ctx, href, codec, request, response, options...)
}
