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
	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/udp/client"
)

// DiscoverDeviceOwnershipHandler receives devices ownership info.
type DiscoverDeviceOwnershipHandler interface {
	Handle(ctx context.Context, client *client.Conn, device doxm.Doxm)
	Error(err error)
}

// DiscoverOwnershipStatus type of discover ownership status.
type DiscoverOwnershipStatus int

const (
	// DiscoverAllDevices discovers owned and disowned devices.
	DiscoverAllDevices = DiscoverOwnershipStatus(0)
	// DiscoverOwnedDevices discovers owned devices,
	DiscoverOwnedDevices = DiscoverOwnershipStatus(1)
	// DiscoverDisownedDevices discovers disowned devices,
	DiscoverDisownedDevices = DiscoverOwnershipStatus(2)
)

// DiscoverDeviceOwnership discovers devices using a CoAP multicast request via UDP.
// It waits for device responses until the context is canceled.
func DiscoverDeviceOwnership(
	ctx context.Context,
	conn []*DiscoveryClient,
	status DiscoverOwnershipStatus,
	handler DiscoverDeviceOwnershipHandler,
) error {
	var opt coap.OptionFunc
	switch status {
	case DiscoverAllDevices:
		return Discover(ctx, conn, doxm.ResourceURI, handleDiscoverOwnershipResponse(ctx, handler))
	case DiscoverOwnedDevices:
		opt = func(m message.Options) message.Options {
			buf := make([]byte, 16)
			opts, _, _ := m.AddString(buf, message.URIQuery, "Owned=TRUE")
			return opts
		}
	case DiscoverDisownedDevices:
		opt = func(m message.Options) message.Options {
			buf := make([]byte, 16)
			opts, _, _ := m.AddString(buf, message.URIQuery, "Owned=FALSE")
			return opts
		}
	default:
		return MakeUnimplemented(fmt.Errorf("unsupported DiscoverOwnershipStatus(%v)", status))
	}

	return Discover(ctx, conn, doxm.ResourceURI, handleDiscoverOwnershipResponse(ctx, handler), opt)
}

func handleDiscoverOwnershipResponse(ctx context.Context, handler DiscoverDeviceOwnershipHandler) func(client *client.Conn, req *pool.Message) {
	return func(client *client.Conn, r *pool.Message) {
		req := r
		if req.Code() != codes.Content {
			handler.Error(fmt.Errorf("request failed: %s", ocf.Dump(req)))
			return
		}

		var doxm doxm.Doxm
		var codec ocf.VNDOCFCBORCodec
		err := codec.Decode(req, &doxm)
		if err != nil {
			handler.Error(fmt.Errorf("decoding %v failed: %w", ocf.DumpHeader(req), err))
			return
		}
		handler.Handle(ctx, client, doxm)
	}
}
