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

	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/go-coap/v3/udp/client"
)

// OwnershipHandler conveys device ownership and errors during discovery.
type OwnershipHandler interface {
	// Handle gets a device ownership.
	Handle(ctx context.Context, doxm doxm.Doxm)
	// Error gets errors during discovery.
	Error(err error)
}

// GetOwnerships discovers device's ownerships using a CoAP multicast request via UDP.
// It waits for device responses until the context is canceled.
func (c *Client) GetOwnerships(
	ctx context.Context,
	discoveryConfiguration DiscoveryConfiguration,
	status DiscoverOwnershipStatus,
	handler OwnershipHandler,
) error {
	multicastConn, err := DialDiscoveryAddresses(ctx, discoveryConfiguration, func(err error) { c.logger.Debug(err.Error()) })
	if err != nil {
		return MakeInvalidArgument(fmt.Errorf("could not get the ownerships: %w", err))
	}
	defer func() {
		for _, conn := range multicastConn {
			if errC := conn.Close(); errC != nil {
				c.logger.Debug(fmt.Errorf("get ownership error: cannot close connection(%s): %w", conn.mcastaddr, errC).Error())
			}
		}
	}()
	h := newOwnershipHandler(handler)
	return DiscoverDeviceOwnership(ctx, multicastConn, status, h)
}

func newOwnershipHandler(
	h OwnershipHandler,
) *ownershipHandler {
	return &ownershipHandler{
		handler: h,
	}
}

type ownershipHandler struct {
	handler OwnershipHandler
}

func (h *ownershipHandler) Handle(ctx context.Context, conn *client.Conn, doxm doxm.Doxm) {
	if errC := conn.Close(); errC != nil {
		h.Error(fmt.Errorf("ownership handler cannot close connection: %w", errC))
	}
	h.handler.Handle(ctx, doxm)
}

func (h *ownershipHandler) Error(err error) {
	h.handler.Error(err)
}
