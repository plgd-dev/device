package local

import (
	"context"
	"fmt"

	"github.com/go-ocf/kit/codec/ocf"
	"github.com/go-ocf/kit/net/coap"
)

// GetResource queries a device for a resource value in CBOR.
func (d *Device) GetResource(
	ctx context.Context,
	href string,
	response interface{},
	options ...coap.OptionFunc,
) error {
	codec := ocf.VNDOCFCBORCodec{}
	return d.GetResourceWithCodec(ctx, href, codec, response, options...)
}

// GetResourceWithCodec queries a device for a resource value.
func (d *Device) GetResourceWithCodec(
	ctx context.Context,
	href string,
	codec coap.Codec,
	response interface{},
	options ...coap.OptionFunc,
) error {
	options = append(options, coap.WithAccept(codec.ContentFormat()))
	client, err := d.connect(ctx, href)
	if err != nil {
		return fmt.Errorf("cannot get resource with href %v: %v", href, err)
	}

	return operationWithRetries(ctx, d.retryFuncFactory, d.retrieveTimeout, func(ctx context.Context) error {
		return client.GetResourceWithCodec(ctx, href, codec, response, options...)
	})
}

// GetSingleResource queries a resource of a given resource type.
// Only a single instance of this resource type is expected.
func (d *Device) GetSingleResource(ctx context.Context, value interface{}, resourceType string) error {
	it, err := d.GetResources(ctx, resourceType)
	if err != nil {
		return fmt.Errorf("canno get resources for %s %+v: %v", d.DeviceID(), resourceType, err)
	}
	ok := it.Next(ctx, value)
	if !ok {
		if it.Err != nil {
			return fmt.Errorf("resource not found for %s %+v: %v", d.DeviceID(), resourceType, it.Err)
		}
		return fmt.Errorf("resource not found for %s %+v", d.DeviceID(), resourceType)
	}
	if it.Next(ctx, value) {
		return fmt.Errorf("too many resource links for %s %+v", d.DeviceID(), resourceType)
	}
	return it.Err
}

// GetResources resolves URIs and returns an iterator for querying resources of given resource types.
func (d *Device) GetResources(ctx context.Context, resourceTypes ...string) (*ResourceIterator, error) {
	links, err := d.GetResourceLinks(ctx)
	if err != nil {
		return nil, err
	}
	return &ResourceIterator{
		device: d,
		hrefs:  links.GetResourceHrefs(resourceTypes...),
	}, nil
}

// ResourceIterator queries resource values.
type ResourceIterator struct {
	Err    error
	hrefs  []string
	i      int
	device *Device
}

// Next queries the next resource value.
// Returns false when failed or having no more items.
// Check it.Err for errors.
func (it *ResourceIterator) Next(ctx context.Context, v interface{}) bool {
	if it.i >= len(it.hrefs) {
		return false
	}

	err := it.device.GetResource(ctx, it.hrefs[it.i], v)
	if err != nil {
		it.Err = fmt.Errorf("could not get a resource value for the device %s: %v", it.device.DeviceID(), err)
		return false
	}

	it.i++
	return true
}
