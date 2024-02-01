/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/bridge/resources/discovery"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/schema"
	plgdResources "github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
)

type Device interface {
	Init()  // start all goroutines for device
	Close() // stop all goroutines for device

	ExportConfig() device.Config // export device config

	GetID() uuid.UUID
	GetLinks(request *net.Request) (links schema.ResourceLinks)
	GetLinksFilteredBy(endpoints schema.Endpoints, deviceIDfilter uuid.UUID, resourceTypesFilter []string, policyBitMaskFitler schema.BitMask) (links schema.ResourceLinks)
	GetName() string
	GetProtocolIndependentID() uuid.UUID
	GetResourceTypes() []string

	HandleRequest(req *net.Request) (*pool.Message, error)

	Range(f func(key string, resource device.Resource) bool)
	AddResource(resource device.Resource)
	LoadAndDeleteResource(resourceHref string) (device.Resource, bool)
	CloseAndDeleteResource(resourceHref string) bool
	GetResource(resourceHref string) (device.Resource, bool)

	GetCloudManager() *cloud.Manager
}

type Service struct {
	cfg                Config
	net                *net.Net
	devices            *coapSync.Map[uuid.UUID, Device]
	done               chan struct{}
	onDiscoveryDevices func(req *net.Request)
}

func (c *Service) LoadDevice(di uuid.UUID) (Device, error) {
	d, ok := c.devices.Load(di)
	if !ok {
		return d, fmt.Errorf("invalid queries: device with di %v not found", di)
	}
	return d, nil
}

func (c *Service) handleDiscoverAllLinks(req *net.Request) (*pool.Message, error) {
	if req.Message.Type() != message.Acknowledgement && req.Message.Type() != message.Reset {
		// discovery is only allowed for CON, NON, UNSET messages
		c.onDiscoveryDevices(req)
	}
	res := discovery.New(plgdResources.ResourceURI, func(request *net.Request) schema.ResourceLinks {
		links := make(schema.ResourceLinks, 0, c.devices.Length()+1)
		for _, d := range c.devices.CopyData() {
			dlinks := d.GetLinks(req)
			if len(dlinks) > 0 {
				links = append(links, dlinks...)
			}
		}
		return links
	})
	defer res.Close()
	return res.Get(req)
}

func (c *Service) DefaultRequestHandler(req *net.Request) (*pool.Message, error) {
	uriPath := req.URIPath()
	if uriPath == "" {
		return nil, nil //nolint:nilnil
	}
	if req.Code() == codes.GET && uriPath == "/.well-known/core" {
		// ignore well-known/core
		return nil, nil //nolint:nilnil
	}
	deviceID := req.DeviceID()
	if req.Code() == codes.GET && uriPath == plgdResources.ResourceURI && deviceID == uuid.Nil {
		return c.handleDiscoverAllLinks(req)
	}
	if deviceID == uuid.Nil {
		return nil, fmt.Errorf("invalid queries: di query is not set")
	}
	d, err := c.LoadDevice(deviceID) // check if device exists et
	if err != nil {
		if req.ControlMessage().Dst.IsMulticast() {
			return nil, nil //nolint:nilnil
		}
		return nil, err
	}
	return d.HandleRequest(req)
}

func New(cfg Config, opts ...Option) (*Service, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}

	o := OptionsCfg{
		onDiscoveryDevices: func(req *net.Request) {
			// nothing to do
		},
		logger: core.NewNilLogger(),
	}
	for _, opt := range opts {
		opt(&o)
	}

	c := Service{
		cfg:                cfg,
		devices:            coapSync.NewMap[uuid.UUID, Device](),
		onDiscoveryDevices: o.onDiscoveryDevices,
		done:               make(chan struct{}),
	}
	n, err := net.New(cfg.API.CoAP.Config, c.DefaultRequestHandler, o.logger)
	if err != nil {
		return nil, err
	}
	c.net = n

	return &c, nil
}

func (c *Service) Serve() error {
	defer close(c.done)
	return c.net.Serve()
}

func (c *Service) Shutdown() error {
	err := c.net.Close()
	if err != nil {
		return err
	}
	<-c.done
	devices := c.devices.LoadAndDeleteAll()
	for _, d := range devices {
		d.Close()
	}
	return nil
}

type NewDeviceFunc func(id uuid.UUID, piid uuid.UUID) (Device, error)

func (c *Service) CreateDevice(id uuid.UUID, newDevice NewDeviceFunc) (Device, error) {
	var d Device
	var err error
	_, oldLoaded := c.devices.ReplaceWithFunc(id, func(oldValue Device, oldLoaded bool) (newValue Device, doDelete bool) {
		if oldLoaded {
			return oldValue, false
		}
		d, err = newDevice(id, resources.ToUUID(c.cfg.API.CoAP.ID))
		if err != nil {
			if oldLoaded {
				return oldValue, false
			}
			return nil, true
		}
		return d, false
	})
	if err != nil {
		return nil, err
	}
	if oldLoaded {
		return nil, fmt.Errorf("device with id %v already exists", id)
	}
	return d, nil
}

func (c *Service) GetOrCreateDevice(id uuid.UUID, newDevice NewDeviceFunc) (d Device, loaded bool, err error) {
	oldDevice, oldLoaded := c.devices.ReplaceWithFunc(id, func(oldValue Device, oldLoaded bool) (newValue Device, doDelete bool) {
		if oldLoaded {
			return oldValue, false
		}
		d, err = newDevice(id, resources.ToUUID(c.cfg.API.CoAP.ID))
		if err != nil {
			if oldLoaded {
				return oldValue, false
			}
			return nil, true
		}
		return d, false
	})
	if err != nil {
		return nil, false, err
	}
	if oldLoaded {
		return oldDevice, true, nil
	}
	return d, false, nil
}

func (c *Service) GetDevice(id uuid.UUID) (Device, bool) {
	return c.devices.Load(id)
}

func (c *Service) CopyDevices() map[uuid.UUID]Device {
	return c.devices.CopyData()
}

func (c *Service) Range(f func(key uuid.UUID, value Device) bool) {
	c.devices.Range(f)
}

func (c *Service) RangeWithLock(f func(key uuid.UUID, value Device) bool) {
	c.devices.Range2(f)
}

func (c *Service) Length() int {
	return c.devices.Length()
}

func (c *Service) GetAndDeleteDevice(id uuid.UUID) (Device, bool) {
	return c.devices.LoadAndDelete(id)
}

func (c *Service) DeleteAndCloseDevice(id uuid.UUID) bool {
	d, ok := c.devices.LoadAndDelete(id)
	if ok {
		d.Close()
	}
	return ok
}
