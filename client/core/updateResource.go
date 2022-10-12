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

func (d *Device) UpdateResource(
	ctx context.Context,
	link schema.ResourceLink,
	request interface{},
	response interface{},
	options ...coap.OptionFunc,
) error {
	codec := ocf.VNDOCFCBORCodec{}
	return d.UpdateResourceWithCodec(ctx, link, codec, request, response, options...)
}

func (d *Device) UpdateResourceWithCodec(
	ctx context.Context,
	link schema.ResourceLink,
	codec coap.Codec,
	request interface{},
	response interface{},
	options ...coap.OptionFunc,
) error {
	_, client, err := d.connectToEndpoints(ctx, link.GetEndpoints())
	if err != nil {
		return MakeInternal(fmt.Errorf("cannot update resource %v: %w", link.Href, err))
	}
	options = append(options, coap.WithAccept(codec.ContentFormat()))

	return client.UpdateResourceWithCodec(ctx, link.Href, codec, request, response, options...)
}
