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

package client

import (
	"context"

	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/resources"
)

type DeviceResourcesObservationHandler = interface {
	Handle(ctx context.Context, links schema.ResourceLinks)
	OnClose()
	Error(err error)
}

type deviceResourcesObserver struct {
	c       *Client
	handler DeviceResourcesObservationHandler
}

func newDeviceResourcesObserver(ctx context.Context, c *Client, deviceID string, handler DeviceResourcesObservationHandler) (string, error) {
	observationID, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, &deviceResourcesObserver{
		c:       c,
		handler: handler,
	})
	if err != nil {
		return "", err
	}
	return observationID, nil
}

func (o *deviceResourcesObserver) Error(err error) {
	o.handler.Error(err)
}

func (o *deviceResourcesObserver) OnClose() {
	o.handler.OnClose()
}

func (o *deviceResourcesObserver) Handle(ctx context.Context, body coap.DecodeFunc) {
	var links schema.ResourceLinks
	err := body(&links)
	if err != nil {
		o.handler.Error(err)
		return
	}
	o.handler.Handle(ctx, links)
}

// ObserveDeviceResources method starts observing links in the device.
// In the absence of a cached device, it is found through multicast and stored with an expiration time.
func (c *Client) ObserveDeviceResources(ctx context.Context, deviceID string, handler DeviceResourcesObservationHandler) (string, error) {
	observationID, err := newDeviceResourcesObserver(ctx, c, deviceID, handler)
	if err != nil {
		return "", err
	}
	return observationID, nil
}

// ObserveDeviceResources method stops observing links in the device.
// In the absence of a cached device, it is found through multicast and stored with an expiration time.
func (c *Client) StopObservingDeviceResources(ctx context.Context, observationID string) (bool, error) {
	return c.StopObservingResource(ctx, observationID)
}
