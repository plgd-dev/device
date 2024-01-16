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

	Range(f func(key string, resource *resources.Resource) bool)
	AddResource(resource *resources.Resource)
	RemoveResource(resource *resources.Resource)
	GetResource(key string) (*resources.Resource, bool)

	UnregisterFromCloud() // unregister device from cloud
}

type Service[D Device] struct {
	cfg                Config
	net                *net.Net
	devices            *coapSync.Map[uuid.UUID, D]
	done               chan struct{}
	onUpdateDevice     func(D)
	onDiscoveryDevices func(req *net.Request)
}

func (c *Service[D]) LoadDevice(di uuid.UUID) (D, error) {
	d, ok := c.devices.Load(di)
	if !ok {
		var d D
		return d, fmt.Errorf("invalid queries: device with di %v not found", di)
	}
	return d, nil
}

func (c *Service[D]) handleDiscoverAllLinks(req *net.Request) (*pool.Message, error) {
	c.onDiscoveryDevices(req)
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

func (c *Service[D]) DefaultRequestHandler(req *net.Request) (*pool.Message, error) {
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

type OptionsCfg[D Device] struct {
	OnUpdateDevice     func(D)
	OnDiscoveryDevices func(req *net.Request)
}

func WithOnUpdateDevice[D Device](f func(D)) Option[D] {
	return func(o *OptionsCfg[D]) {
		if f != nil {
			o.OnUpdateDevice = f
		}
	}
}

func WithOnDiscoveryDevices[D Device](f func(req *net.Request)) Option[D] {
	return func(o *OptionsCfg[D]) {
		if f != nil {
			o.OnDiscoveryDevices = f
		}
	}
}

type Option[D Device] func(*OptionsCfg[D])

func New[D Device](cfg Config, opts ...Option[D]) (*Service[D], error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}

	o := OptionsCfg[D]{
		OnUpdateDevice: func(d D) {
			// nothing to do
		},
		OnDiscoveryDevices: func(req *net.Request) {
			// nothing to do
		},
	}
	for _, opt := range opts {
		opt(&o)
	}

	c := Service[D]{
		cfg:                cfg,
		devices:            coapSync.NewMap[uuid.UUID, D](),
		onUpdateDevice:     o.OnUpdateDevice,
		onDiscoveryDevices: o.OnDiscoveryDevices,
	}
	n, err := net.New(cfg.API.CoAP.Config, c.DefaultRequestHandler)
	if err != nil {
		return nil, err
	}
	c.net = n

	return &c, nil
}

func (c *Service[D]) Serve() error {
	defer close(c.done)
	return c.net.Serve()
}

func (c *Service[D]) CreateDevice(id uuid.UUID, name string, newDevice func(id uuid.UUID, name string, piid uuid.UUID, onUpdateDevice func(D)) D) (D, bool) {
	var d D
	_, oldLoaded := c.devices.ReplaceWithFunc(id, func(oldValue D, oldLoaded bool) (newValue D, doDelete bool) {
		if oldLoaded {
			return oldValue, false
		}
		d = newDevice(id, name, resources.ToUUID(c.cfg.API.CoAP.ID), c.onUpdateDevice)
		return d, false
	})
	if oldLoaded {
		return d, false
	}
	return d, true
}

func (c *Service[D]) GetOrCreateDevice(id uuid.UUID, name string, newDevice func(id uuid.UUID, name string, piid uuid.UUID, onUpdateDevice func(D)) D) (d D, loaded bool) {
	return c.devices.LoadOrStoreWithFunc(id, func(value D) D {
		return value
	}, func() D {
		return newDevice(id, name, resources.ToUUID(c.cfg.API.CoAP.ID), c.onUpdateDevice)
	})
}

func (c *Service[D]) GetDevice(id uuid.UUID) (D, bool) {
	return c.devices.Load(id)
}

func (c *Service[D]) CopyDevices() map[uuid.UUID]D {
	return c.devices.CopyData()
}

func (c *Service[D]) Range(f func(key uuid.UUID, value D) bool) {
	c.devices.Range(f)
}

func (c *Service[D]) RangeWithLock(f func(key uuid.UUID, value D) bool) {
	c.devices.Range2(f)
}

func (c *Service[D]) Length() int {
	return c.devices.Length()
}

func (c *Service[D]) GetAndDeleteDevice(id uuid.UUID) (D, bool) {
	return c.devices.LoadAndDelete(id)
}

func (c *Service[D]) DeleteAndCloseDevice(id uuid.UUID) {
	d, ok := c.devices.LoadAndDelete(id)
	if ok {
		d.Close()
	}
}
