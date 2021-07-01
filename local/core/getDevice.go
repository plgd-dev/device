package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/go-coap/v2/udp/client"
	"github.com/plgd-dev/sdk/pkg/net/coap"
	"github.com/plgd-dev/sdk/schema"
)

// GetDevice performs a multicast and returns a device object if the device responds.
func (c *Client) GetDevice(ctx context.Context, discoveryConfiguration DiscoveryConfiguration, deviceID string) (*Device, error) {
	findCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	multicastConn := DialDiscoveryAddresses(findCtx, discoveryConfiguration, c.errFunc)
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()

	h := newDeviceHandler(c.getDeviceConfiguration(), deviceID, cancel)
	// we want to just get "oic.wk.d" resource, because links will be get via unicast to /oic/res
	err := DiscoverDevices(findCtx, multicastConn, h, coap.WithResourceType("oic.wk.d"))
	if err != nil {
		return nil, MakeDataLoss(fmt.Errorf("could not get the device %s: %w", deviceID, err))
	}
	d := h.Device()
	if d == nil {
		return nil, MakeInternal(fmt.Errorf("no response from the device %s", deviceID))
	}

	return d, nil
}

func newDeviceHandler(
	deviceCfg deviceConfiguration,
	deviceID string,
	cancel context.CancelFunc,
) *deviceHandler {
	return &deviceHandler{
		deviceCfg: deviceCfg,
		deviceID:  deviceID,
		cancel:    cancel,
	}
}

type deviceHandler struct {
	deviceCfg deviceConfiguration
	deviceID  string
	cancel    context.CancelFunc

	lock   sync.Mutex
	device *Device
	err    error
}

func (h *deviceHandler) Device() *Device {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.device
}

func (h *deviceHandler) Handle(ctx context.Context, conn *client.ClientConn, links schema.ResourceLinks) {
	conn.Close()
	h.lock.Lock()
	defer h.lock.Unlock()

	link, err := GetResourceLink(links, "/oic/d")
	if err != nil {
		h.err = err
		return
	}
	deviceID := link.GetDeviceID()
	if deviceID == "" {
		h.err = MakeInternal(fmt.Errorf("cannot determine deviceID"))
		return
	}

	if h.device != nil || deviceID != h.deviceID {
		return
	}
	if len(link.ResourceTypes) == 0 {
		h.err = MakeDataLoss(fmt.Errorf("cannot get resource types for %v: is empty", deviceID))
		return
	}
	d := NewDevice(h.deviceCfg, deviceID, link.ResourceTypes, link.GetEndpoints())

	h.device = d
	h.cancel()
}

func (h *deviceHandler) Error(err error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.err = err
}

func (h *deviceHandler) Err() error {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.err
}
