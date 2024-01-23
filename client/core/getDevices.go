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
	"sync"

	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/go-coap/v3/udp/client"
)

// DeviceHandler conveys device connections and errors during discovery.
type DeviceHandler interface {
	// Handle gets a device connection and is responsible for closing it.
	Handle(ctx context.Context, device *Device, deviceLinks schema.ResourceLinks)
	// Error gets errors during discovery.
	Error(err error)
}

// DeviceMulticastHandler conveys device connections and errors during discovery.
type DeviceMulticastHandler interface {
	// Handle gets a device connection and is responsible for closing it.
	Handle(ctx context.Context, device *Device)
	// Error gets errors during discovery.
	Error(err error)
}

// GetDevicesByMulticast discovers devices using a CoAP multicast request via UDP.
// Device resources can be queried in DeviceHandler using device.Client,
func (c *Client) GetDevicesByMulticast(ctx context.Context, discoveryConfiguration DiscoveryConfiguration, handler DeviceMulticastHandler) error {
	multicastConn, err := DialDiscoveryAddresses(ctx, discoveryConfiguration, func(err error) { c.logger.Debug(err.Error()) })
	if err != nil {
		return MakeInvalidArgument(fmt.Errorf("could not get the devices: %w", err))
	}
	defer func() {
		for _, conn := range multicastConn {
			if errC := conn.Close(); errC != nil {
				c.logger.Debug(fmt.Errorf("get devices error: cannot close connection(%s): %w", conn.mcastaddr, errC).Error())
			}
		}
	}()
	// we want to just get "oic.wk.d" resource, because links will be get via unicast to /oic/res
	return DiscoverDevices(ctx, multicastConn, newDiscoveryHandler(c.getDeviceConfiguration(), handler), coap.WithResourceType(device.ResourceType))
}

func newDiscoveryHandler(
	deviceCfg DeviceConfiguration,
	h DeviceMulticastHandler,
) *discoveryHandler {
	return &discoveryHandler{
		deviceCfg: deviceCfg,
		handler:   h,
	}
}

type discoveryHandler struct {
	deviceCfg               DeviceConfiguration
	handler                 DeviceMulticastHandler
	filterDiscoveredDevices sync.Map
}

func (h *discoveryHandler) Handle(ctx context.Context, conn *client.Conn, links schema.ResourceLinks) {
	if errC := conn.Close(); errC != nil {
		h.handler.Error(fmt.Errorf("discovery handler cannot close connection: %w", errC))
	}
	deviceLinks := make(map[string]schema.ResourceLinks)
	for _, link := range links {
		deviceID := link.GetDeviceID()
		if deviceID == "" {
			h.handler.Error(fmt.Errorf("cannot determine deviceID"))
			continue
		}
		deviceLinks[deviceID] = append(deviceLinks[deviceID], link)
	}
	for deviceID, links := range deviceLinks {
		link, err := GetResourceLink(links, device.ResourceURI)
		if err != nil {
			h.handler.Error(err)
			continue
		}
		if len(link.ResourceTypes) == 0 {
			h.handler.Error(fmt.Errorf("cannot get resource types for %v: is empty", deviceID))
			continue
		}
		_, loaded := h.filterDiscoveredDevices.LoadOrStore(deviceID, true)
		if loaded {
			continue
		}
		d := NewDevice(h.deviceCfg, deviceID, link.ResourceTypes, link.GetEndpoints)
		h.handler.Handle(ctx, d)
	}
}

func (h *discoveryHandler) Error(err error) {
	h.handler.Error(err)
}
