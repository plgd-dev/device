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
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/udp/client"
	"github.com/plgd-dev/kit/v2/net"
)

// DiscoverDevicesHandler receives device links and errors from the discovery multicast request.
type DiscoverDevicesHandler interface {
	Handle(ctx context.Context, client *client.Conn, device schema.ResourceLinks)
	Error(err error)
}

// DiscoverDevices discovers devices using a CoAP multicast request via UDP.
// It waits for device responses until the context is canceled.
// Device resources can be queried in DiscoveryHandler.
// An empty typeFilter queries all resource types.
// Note: Iotivity 1.3 which responds with BadRequest if more than 1 resource type is queried.
func DiscoverDevices(
	ctx context.Context,
	conn []*DiscoveryClient,
	handler DiscoverDevicesHandler,
	options ...coap.OptionFunc,
) error {
	options = append(options, coap.WithAccept(message.AppOcfCbor))
	return Discover(ctx, conn, resources.ResourceURI, handleResponse(ctx, handler), options...)
}

func handleResponse(ctx context.Context, handler DiscoverDevicesHandler) func(*client.Conn, *pool.Message) {
	return func(cc *client.Conn, r *pool.Message) {
		req := r
		if req.Code() != codes.Content {
			handler.Error(fmt.Errorf("request failed: %s", ocf.Dump(req)))
			return
		}

		var links schema.ResourceLinks
		var codec DiscoverDeviceCodec

		err := codec.Decode(req, &links)
		if err != nil {
			handler.Error(fmt.Errorf("decoding %v failed: %w", ocf.DumpHeader(req), err))
			return
		}
		addr, err := net.Parse(string(schema.UDPScheme), cc.RemoteAddr())
		if err != nil {
			handler.Error(fmt.Errorf("invalid address %v: %w", cc.RemoteAddr(), err))
			return
		}
		links = links.PatchEndpoint(addr, nil)
		if len(links) > 0 {
			handler.Handle(ctx, cc, links)
		}
	}
}
