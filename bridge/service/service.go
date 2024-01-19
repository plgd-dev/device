package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/bridge/resources/discovery"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/interfaces"
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

	UnregisterFromCloud() // unregister device from cloud
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

		links = append(links, schema.ResourceLink{
			Href:          plgdResources.ResourceURI,
			ResourceTypes: []string{plgdResources.ResourceType},
			Interfaces:    []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_R},
			Endpoints:     req.Endpoints,
			Anchor:        "ocf://" + resources.ToUUID(c.cfg.API.CoAP.ID).String(),
			DeviceID:      resources.ToUUID(c.cfg.API.CoAP.ID).String(),
			Policy: &schema.Policy{
				BitMask: schema.Discoverable,
			},
		})
		return links
	})
	defer res.Close()
	return res.Get(req)
}

func (c *Service) DefaultRequestHandler(req *net.Request) (*pool.Message, error) {
	uriPath := req.URIPath()
	if uriPath == "" {
		return nil, nil
	}
	if req.Code() == codes.GET && uriPath == "/.well-known/core" {
		// ignore well-known/core
		return nil, nil
	}
	if req.Code() == codes.GET && uriPath == plgdResources.ResourceURI {
		return c.handleDiscoverAllLinks(req)
	}
	if req.DeviceID() == uuid.Nil {
		return nil, fmt.Errorf("invalid queries: di query is not set")
	}
	d, err := c.LoadDevice(req.DeviceID()) // check if device exists et
	if err != nil {
		return nil, err
	}
	return d.HandleRequest(req)
}

type OptionsCfg struct {
	OnDiscoveryDevices func(req *net.Request)
}

func WithOnDiscoveryDevices(f func(req *net.Request)) Option {
	return func(o *OptionsCfg) {
		if f != nil {
			o.OnDiscoveryDevices = f
		}
	}
}

type Option func(*OptionsCfg)

func New(cfg Config, opts ...Option) (*Service, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}

	o := OptionsCfg{
		OnDiscoveryDevices: func(req *net.Request) {
			// nothing to do
		},
	}
	for _, opt := range opts {
		opt(&o)
	}

	c := Service{
		cfg:                cfg,
		devices:            coapSync.NewMap[uuid.UUID, Device](),
		onDiscoveryDevices: o.OnDiscoveryDevices,
	}
	n, err := net.New(cfg.API.CoAP.Config, c.DefaultRequestHandler)
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
	devices := c.devices.LoadAndDeleteAll()
	for _, d := range devices {
		d.Close()
	}
	return c.net.Close()
}

type NewDeviceFunc func(id uuid.UUID, piid uuid.UUID) Device

func (c *Service) CreateDevice(id uuid.UUID, newDevice NewDeviceFunc) (Device, bool) {
	var d Device
	_, oldLoaded := c.devices.ReplaceWithFunc(id, func(oldValue Device, oldLoaded bool) (newValue Device, doDelete bool) {
		if oldLoaded {
			return oldValue, false
		}
		d = newDevice(id, resources.ToUUID(c.cfg.API.CoAP.ID))
		return d, false
	})
	if oldLoaded {
		return d, false
	}
	return d, true
}

func (c *Service) GetOrCreateDevice(id uuid.UUID, newDevice NewDeviceFunc) (d Device, loaded bool) {
	return c.devices.LoadOrStoreWithFunc(id, func(value Device) Device {
		return value
	}, func() Device {
		return newDevice(id, resources.ToUUID(c.cfg.API.CoAP.ID))
	})
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
