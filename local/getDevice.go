package local

import (
	"context"
	"fmt"
	"sync"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/sdk/schema"
)

// GetDevice performs a multicast and returns a device object if the device responds.
func (c *Client) GetDevice(ctx context.Context, deviceID string) (*Device, schema.ResourceLinks, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	multicastConn := DialDiscoveryAddresses(ctx, c.discoveryConfiguration, c.errFunc)
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()

	h := newDeviceHandler(c.getDeviceConfiguration(), deviceID, cancel)
	err := DiscoverDevices(ctx, multicastConn, h)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get the device %s: %w", deviceID, err)
	}
	d, dlinks := h.Device()
	if d == nil {
		return nil, nil, fmt.Errorf("no response from the device %s", deviceID)
	}
	return d, dlinks, nil
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

	lock        sync.Mutex
	device      *Device
	deviceLinks schema.ResourceLinks
	err         error
}

func (h *deviceHandler) Device() (*Device, schema.ResourceLinks) {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.device, h.deviceLinks
}

func (h *deviceHandler) Handle(ctx context.Context, conn *gocoap.ClientConn, links schema.ResourceLinks) {
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
		h.err = fmt.Errorf("cannot determine deviceID")
		return
	}

	if h.device != nil || deviceID != h.deviceID {
		return
	}
	if len(link.ResourceTypes) == 0 {
		h.err = fmt.Errorf("cannot get resource types for %v: is empty", deviceID)
		return
	}
	d := NewDevice(h.deviceCfg, deviceID, link.ResourceTypes)

	h.device = d
	h.deviceLinks = links
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
