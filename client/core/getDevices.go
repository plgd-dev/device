package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/go-coap/v2/udp/client"
)

// DeviceHandler conveys device connections and errors during discovery.
type DeviceHandler interface {
	// Handle gets a device connection and is responsible for closing it.
	Handle(ctx context.Context, device *Device, deviceLinks schema.ResourceLinks)
	// Error gets errors during discovery.
	Error(err error)
}

type deprecatedDeviceHandler struct {
	h DeviceHandler
}

func (h deprecatedDeviceHandler) Handle(ctx context.Context, device *Device) {
	eps := device.GetEndpoints()
	links, err := device.GetResourceLinks(ctx, eps)
	if err != nil {
		device.Close(ctx)
		h.Error(err)
		return
	}

	h.h.Handle(ctx, device, links)
}

// Error gets errors during discovery.
func (h deprecatedDeviceHandler) Error(err error) {
	h.h.Error(err)
}

// GetDevices discovers devices using a CoAP multicast request via UDP to default addresses.
// Device resources can be queried in DeviceHandler using device.Client,
// DEPRECATED
func (c *Client) GetDevices(ctx context.Context, handler DeviceHandler) error {
	return c.GetDevicesV2(ctx, DefaultDiscoveryConfiguration(), deprecatedDeviceHandler{handler})
}

// DeviceHandler conveys device connections and errors during discovery.
type DeviceHandlerV2 interface {
	// Handle gets a device connection and is responsible for closing it.
	Handle(ctx context.Context, device *Device)
	// Error gets errors during discovery.
	Error(err error)
}

// GetDevices discovers devices using a CoAP multicast request via UDP.
// Device resources can be queried in DeviceHandler using device.Client,
func (c *Client) GetDevicesV2(ctx context.Context, discoveryConfiguration DiscoveryConfiguration, handler DeviceHandlerV2) error {
	multicastConn, err := DialDiscoveryAddresses(ctx, discoveryConfiguration, c.errFunc)
	if err != nil {
		return MakeInvalidArgument(fmt.Errorf("could not get the devices: %w", err))
	}
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()
	// we want to just get "oic.wk.d" resource, because links will be get via unicast to /oic/res
	return DiscoverDevices(ctx, multicastConn, newDiscoveryHandler(c.getDeviceConfiguration(), handler), coap.WithResourceType(device.ResourceType))
}

func newDiscoveryHandler(
	deviceCfg DeviceConfiguration,
	h DeviceHandlerV2,
) *discoveryHandler {
	return &discoveryHandler{
		deviceCfg: deviceCfg,
		handler:   h,
	}
}

type discoveryHandler struct {
	deviceCfg               DeviceConfiguration
	handler                 DeviceHandlerV2
	filterDiscoveredDevices sync.Map
}

func (h *discoveryHandler) Handle(ctx context.Context, conn *client.ClientConn, links schema.ResourceLinks) {
	conn.Close()
	link, err := GetResourceLink(links, device.ResourceURI)
	if err != nil {
		h.handler.Error(err)
		return
	}
	deviceID := link.GetDeviceID()
	if deviceID == "" {
		h.handler.Error(fmt.Errorf("cannot determine deviceID"))
		return
	}
	if len(link.ResourceTypes) == 0 {
		h.handler.Error(fmt.Errorf("cannot get resource types for %v: is empty", deviceID))
		return
	}
	_, loaded := h.filterDiscoveredDevices.LoadOrStore(deviceID, true)
	if loaded {
		return
	}
	d := NewDevice(h.deviceCfg, deviceID, link.ResourceTypes, link.GetEndpoints)
	h.handler.Handle(ctx, d)
}

func (h *discoveryHandler) Error(err error) {
	h.handler.Error(err)
}
