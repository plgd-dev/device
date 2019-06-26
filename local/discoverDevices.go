package local

import (
	"context"
	"fmt"
	"sync"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/sdk/schema"

	"github.com/go-ocf/sdk/local/device"
	"github.com/go-ocf/sdk/local/resource"
)

type DiscoveredDevice struct {
	DeviceID          string
	DeviceLinks       *schema.DeviceLinks
	OwnershipResource *schema.Doxm
}

type DiscoveredDeviceHandler interface {
	Handle(ctx context.Context, discoveredDevice DiscoveredDevice)
	Error(err error)
}

type syncMapDiscoveredDevices struct {
	devices map[string]*DiscoveredDevice
	lock    sync.Mutex
}

type discoveryDeviceHandler struct {
	data    *syncMapDiscoveredDevices
	handler DiscoveredDeviceHandler
}

func (d *discoveryDeviceHandler) setDeviceLinks(ctx context.Context, deviceID string, deviceLinks schema.DeviceLinks) DiscoveredDevice {
	d.data.lock.Lock()
	defer d.data.lock.Unlock()
	dev, ok := d.data.devices[deviceID]
	if !ok {
		dev = &DiscoveredDevice{
			DeviceID: deviceID,
		}
		d.data.devices[deviceID] = dev
	}
	dev.DeviceLinks = &deviceLinks
	return *dev
}

func (d *discoveryDeviceHandler) Error(err error) {
	d.handler.Error(err)
}

func (d *discoveryDeviceHandler) Handle(ctx context.Context, client *device.Client) {
	res := d.setDeviceLinks(ctx, client.DeviceID(), client.GetDeviceLinks())
	if res.DeviceLinks != nil {
		d.handler.Handle(ctx, res)
	}
}

type discoveryOwnerHandler struct {
	data    *syncMapDiscoveredDevices
	handler DiscoveredDeviceHandler
}

func (d *discoveryOwnerHandler) setOwnership(ctx context.Context, deviceID string, ownRes schema.Doxm) DiscoveredDevice {
	d.data.lock.Lock()
	defer d.data.lock.Unlock()
	dev, ok := d.data.devices[deviceID]
	if !ok {
		dev = &DiscoveredDevice{
			DeviceID: deviceID,
		}
		d.data.devices[deviceID] = dev
	}
	dev.OwnershipResource = &ownRes
	return *dev
}

func (d *discoveryOwnerHandler) Error(err error) {
	d.handler.Error(err)
}

func (d *discoveryOwnerHandler) Handle(ctx context.Context, conn *gocoap.ClientConn, ownRes schema.Doxm) {
	conn.Close()
	d.handler.Handle(ctx, d.setOwnership(ctx, ownRes.DeviceId, ownRes))
}

func (c *Client) DiscoverDevices(ctx context.Context, typeFilter []string, handler DiscoveredDeviceHandler) error {
	data := syncMapDiscoveredDevices{
		devices: make(map[string]*DiscoveredDevice),
	}
	var wg sync.WaitGroup
	wg.Add(2)

	hDev := discoveryDeviceHandler{
		data:    &data,
		handler: handler,
	}

	var errors []error
	var errorsLock sync.Mutex

	go func() {
		defer wg.Done()
		err := c.GetDevices(ctx, typeFilter, &hDev)
		if err != nil {
			errorsLock.Lock()
			errors = append(errors, err)
			defer errorsLock.Unlock()
		}
	}()

	hOwn := discoveryOwnerHandler{
		data:    &data,
		handler: handler,
	}
	go func() {
		defer wg.Done()
		err := c.GetDeviceOwnership(ctx, resource.DiscoverAllDevices, &hOwn)
		if err != nil {
			errorsLock.Lock()
			errors = append(errors, err)
			defer errorsLock.Unlock()
		}
	}()

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}
