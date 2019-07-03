package local

import (
	"context"
	"fmt"
	"sync"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"
)

// GetDevice performs a multicast and returns a device object if the device responds.
func (c *Client) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	h := newDeviceHandler(deviceID, cancel)
	err := resource.DiscoverDevices(ctx, c.conn, h, coap.WithDeviceID(deviceID))
	if err != nil {
		return nil, fmt.Errorf("could not get the device %s: %v", deviceID, err)
	}
	d := h.Device()
	if d == nil {
		return nil, fmt.Errorf("no response from the device %s", deviceID)
	}
	return d, nil
}

func newDeviceHandler(deviceID string, cancel context.CancelFunc) *deviceHandler {
	return &deviceHandler{deviceID: deviceID, cancel: cancel}
}

type deviceHandler struct {
	deviceID string
	cancel   context.CancelFunc

	lock   sync.Mutex
	device *Device
}

func (h *deviceHandler) Device() *Device {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.device
}

func (h *deviceHandler) Handle(ctx context.Context, conn *gocoap.ClientConn, links schema.DeviceLinks) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.device != nil || links.ID != h.deviceID {
		return
	}
	h.device = NewDevice(links, conn)
	h.cancel()
}

func (h *deviceHandler) Error(err error) {
}
