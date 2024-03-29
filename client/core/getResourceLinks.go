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

	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/kit/v2/net"
)

func getResourceLinks(ctx context.Context, addr net.Addr, client *coap.ClientCloseHandler, deviceEndpoints schema.Endpoints, options ...coap.OptionFunc) (schema.ResourceLinks, error) {
	options = append(options, coap.WithAccept(message.AppOcfCbor))
	var links schema.ResourceLinks

	var codec DiscoverDeviceCodec
	err := client.GetResourceWithCodec(ctx, resources.ResourceURI, codec, &links, options...)
	if err != nil {
		return nil, err
	}
	return links.PatchEndpoint(addr, deviceEndpoints), nil
}

func (d *Device) GetResourceLinks(ctx context.Context, endpoints []schema.Endpoint, options ...coap.OptionFunc) (schema.ResourceLinks, error) {
	addr, client, err := d.connectToEndpoints(ctx, endpoints)
	if err != nil {
		return nil, MakeDataLoss(fmt.Errorf("cannot get resource links for %v with endpoints %+v: %w", d.DeviceID(), endpoints, err))
	}
	links, err := getResourceLinks(ctx, addr, client, d.GetEndpoints(), options...)
	if err != nil {
		return links, MakeDataLoss(fmt.Errorf("cannot get resource links for %v: %w", d.DeviceID(), err))
	}
	return links, nil
}

func GetResourceLink(links schema.ResourceLinks, href string) (schema.ResourceLink, error) {
	link, ok := links.GetResourceLink(href)
	if !ok {
		return link, MakeUnavailable(fmt.Errorf("resource \"%v\" not found", href))
	}
	return link, nil
}
