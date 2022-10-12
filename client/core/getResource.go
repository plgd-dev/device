// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package core

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
)

// GetResource queries a device for a resource value in CBOR.
func (d *Device) GetResource(
	ctx context.Context,
	link schema.ResourceLink,
	response interface{},
	options ...coap.OptionFunc,
) error {
	codec := ocf.VNDOCFCBORCodec{}
	return d.GetResourceWithCodec(ctx, link, codec, response, options...)
}

// GetResourceWithCodec queries a device for a resource value.
func (d *Device) GetResourceWithCodec(
	ctx context.Context,
	link schema.ResourceLink,
	codec coap.Codec,
	response interface{},
	options ...coap.OptionFunc,
) error {
	options = append(options, coap.WithAccept(codec.ContentFormat()))
	_, client, err := d.connectToEndpoints(ctx, link.GetEndpoints())
	if err != nil {
		return fmt.Errorf("cannot get resource %v: %w", link.Href, err)
	}
	return client.GetResourceWithCodec(ctx, link.Href, codec, response, options...)
}

// GetResources resolves URIs and returns an iterator for querying resources of given resource types.
func (d *Device) GetResources(ctx context.Context, links schema.ResourceLinks) *ResourceIterator {
	return &ResourceIterator{
		device: d,
		links:  links,
	}
}

// ResourceIterator queries resource values.
type ResourceIterator struct {
	Err    error
	links  schema.ResourceLinks
	i      int
	device *Device
}

// Next queries the next resource value.
// Returns false when failed or having no more items.
// Check it.Err for errors.
// Usage:
//
//	for {
//		var v MyStruct
//		if !it.Next(ctx, &v) {
//			break
//		}
//	}
//	if it.Err != nil {
//	}
func (it *ResourceIterator) Next(ctx context.Context, v interface{}) bool {
	if it.Err != nil || it.i >= len(it.links) {
		return false
	}

	err := it.device.GetResource(ctx, it.links[it.i], v)
	if err != nil {
		it.Err = MakeDataLoss(fmt.Errorf("could not get a resource value for the device %s: %w", it.device.DeviceID(), err))
		return false
	}

	it.i++
	return true
}
