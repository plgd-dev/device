package backend

import (
	"context"

	codecOcf "github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"

	"github.com/go-ocf/grpc-gateway/pb"
)

// WithInterface updates/gets resource with interface directly from a device.
func WithInterface(resourceInterface string) func(opts ResourceOptions) {
	return func(opts ResourceOptions) {
		opts.SetResourceInterface(resourceInterface)
	}
}

// WithSkipShadow gets resource directly from a device without using interface for backend client.
func WithSkipShadow() func(opts ResourceOptions) {
	return func(opts ResourceOptions) {
		opts.SkipShadow()
	}
}

// ResourceOptions collections of options.
type ResourceOptions = interface {
	SetResourceInterface(resource string)
	SkipShadow()
}

// ResourceOption option definition.
type ResourceOption = func(opts ResourceOptions)

type resourceOptions struct {
	resourceInterface string
	skipShadow        bool
}

func (o *resourceOptions) SetResourceInterface(resourceInterface string) {
	o.resourceInterface = resourceInterface
}

func (o *resourceOptions) SkipShadow() {
	o.skipShadow = true
}

// GetResourceWithCodec retrieves content of a resource from the backend.
func (c *Client) GetResourceWithCodec(
	ctx context.Context,
	deviceID string,
	href string,
	codec kitNetCoap.Codec,
	response interface{},
	opts ...ResourceOption,
) error {
	var cfg resourceOptions
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.resourceInterface != "" || cfg.skipShadow {
		return c.getResourceFromDevice(ctx, deviceID, href, cfg.resourceInterface, codec, response)
	}
	return c.getResource(ctx, deviceID, href, codec, response)
}

// GetResource retrieves content of a resource from the backend in OCF-CBOR format.
func (c *Client) GetResource(
	ctx context.Context,
	deviceID string,
	href string,
	response interface{},
	opts ...ResourceOption,
) error {
	var codec codecOcf.VNDOCFCBORCodec
	return c.GetResourceWithCodec(ctx, deviceID, href, codec, response, opts...)
}

// GetResource retrieves content of a resource from the KiConnect backend.
func (c *Client) getResource(
	ctx context.Context,
	deviceID string,
	href string,
	codec kitNetCoap.Codec,
	response interface{}) error {
	var resp *pb.ResourceValue
	err := c.RetrieveResourcesByResourceIDs(ctx, MakeResourceIDCallback(deviceID, href, func(v pb.ResourceValue) {
		resp = &v
	}))
	if err != nil {
		return err
	}

	return DecodeContentWithCodec(codec, resp.GetContent().GetContentType(), resp.GetContent().GetData(), response)
}
