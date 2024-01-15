package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	resourcesDevice "github.com/plgd-dev/device/v2/bridge/resources/device"
	"github.com/plgd-dev/device/v2/bridge/resources/discovery"
	"github.com/plgd-dev/device/v2/schema"
	plgdCloud "github.com/plgd-dev/device/v2/schema/cloud"
	plgdDevice "github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	plgdResources "github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
)

const ControllerFileName = "Service.yaml"

type Service struct {
	cfg                Config
	net                *net.Net
	devices            *coapSync.Map[uuid.UUID, *device.Device] // Plgd device ID -> Plgd device
	done               chan struct{}
	onUpdateDevice     func(*device.Device)
	onDiscoveryDevices func(req *net.Request)
}

func (c *Service) LoadDevice(deviceID string) (*device.Device, error) {
	di, err := uuid.Parse(deviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid queries: di query is not a valid uuid: %v", err)
	}
	d, ok := c.devices.Load(di)
	if !ok {
		return nil, fmt.Errorf("invalid queries: device with di %v not found", di)
	}
	return d, nil
}

func handleDiscoverAllLinks(c *Service, req *net.Request) (*pool.Message, error) {
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
		return handleDiscoverAllLinks(c, req)
	}
	if req.DeviceID() == "" {
		return nil, fmt.Errorf("invalid queries: di query is not set")
	}
	d, err := c.LoadDevice(req.DeviceID()) // check if device exists et
	if err != nil {
		return nil, err
	}
	return d.HandleRequest(req)
}

type OptionsCfg struct {
	OnUpdateDevice     func(*device.Device)
	OnDiscoveryDevices func(req *net.Request)
}

func WithOnUpdateDevice(f func(*device.Device)) Option {
	return func(o *OptionsCfg) {
		if f != nil {
			o.OnUpdateDevice = f
		}
	}
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
		OnUpdateDevice: func(d *device.Device) {
			// nothing to do
		},
		OnDiscoveryDevices: func(req *net.Request) {
			// nothing to do
		},
	}
	for _, opt := range opts {
		opt(&o)
	}

	c := Service{
		cfg:                cfg,
		devices:            coapSync.NewMap[uuid.UUID, *device.Device](),
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

func (c *Service) Serve() error {
	defer close(c.done)
	return c.net.Serve()
}

type DeviceOptions struct {
	DeviceTypes        []string
	EnableCloudManager bool
	DeviceData         any
}

type DeviceOption func(*DeviceOptions)

func WithDeviceTypes(deviceTypes []string) DeviceOption {
	return func(o *DeviceOptions) {
		if deviceTypes != nil {
			o.DeviceTypes = deviceTypes
		}
	}
}

func WithDeviceData(deviceData any) DeviceOption {
	return func(o *DeviceOptions) {
		if deviceData != nil {
			o.DeviceData = deviceData
		}
	}
}

func WithEnableCloudManager(enable bool) DeviceOption {
	return func(o *DeviceOptions) {
		o.EnableCloudManager = enable
	}
}

func (c *Service) newDevice(deviceID uuid.UUID, name string, opt ...DeviceOption) *device.Device {
	opts := DeviceOptions{
		EnableCloudManager: true,
	}
	for _, o := range opt {
		o(&opts)
	}
	d := device.New(device.Config{
		Name:                  name,
		ResourceTypes:         opts.DeviceTypes,
		ID:                    deviceID.String(),
		ProtocolIndependentID: resources.ToUUID(c.cfg.API.CoAP.ID).String(),
	}, c.onUpdateDevice, opts.DeviceData)
	d.AddResource(resourcesDevice.New(plgdDevice.ResourceURI, d).Resource)
	if opts.EnableCloudManager {
		d.SetCloudManager(plgdCloud.ResourceURI, c.DefaultRequestHandler, c.cfg.API.CoAP.MaxMessageSize)
	}
	return d
}

func (c *Service) AddDevice(id uuid.UUID, name string, opt ...DeviceOption) *device.Device {
	d := c.newDevice(id, name, opt...)
	old, ok := c.devices.Replace(id, d)
	if ok {
		old.Close()
	}
	return d
}

func (c *Service) GetDevice(id uuid.UUID) (*device.Device, bool) {
	return c.devices.Load(id)
}

func (c *Service) RemoveDevice(id uuid.UUID) {
	d, ok := c.devices.LoadAndDelete(id)
	if ok {
		d.Close()
	}
}
