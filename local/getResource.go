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
	return d.connection("TODO").GetResourceWithCodec(ctx, href, codec, response, options...)
}

// GetSingleResource queries a resource of a given resource type.
// Only a single instance of this resource type is expected.
func (d *Device) GetSingleResource(ctx context.Context, value interface{}, resourceType string) error {
	it := d.GetResources(resourceType)
	ok := it.Next(ctx, value)
	if !ok {
		return fmt.Errorf("resource not found for %s %+v", d.ID, resourceType)
	}
	if it.Next(ctx, value) {
		return fmt.Errorf("too many resource links for %s %+v", d.ID, resourceType)
	}
	return it.Err
}

// GetResources resolves URIs and returns an iterator for querying resources of given resource types.
func (d *Device) GetResources(resourceTypes ...string) *ResourceIterator {
	return &ResourceIterator{
		device: d,
		hrefs:  d.GetResourceHrefs(resourceTypes...),
	}
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
		it.Err = fmt.Errorf("could not get a resource value for the device %s: %v", it.device.ID, err)
		return false
	}

	it.i++
	return true
}
