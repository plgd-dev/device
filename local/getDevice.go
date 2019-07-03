package local

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/local/device"
	"github.com/go-ocf/sdk/local/resource"
)

type deviceHandler struct {
	deviceID string
	cancel   context.CancelFunc

	client *device.Client
	lock   sync.Mutex
	err    error
}

func newDeviceHandler(deviceID string, cancel context.CancelFunc) *deviceHandler {
	return &deviceHandler{deviceID: deviceID, cancel: cancel}
}

func (h *deviceHandler) Handle(ctx context.Context, client *device.Client) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if client.DeviceID() == h.deviceID {
		h.client = client
		h.cancel()
	}
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

func (h *deviceHandler) Client() *device.Client {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.client
}

// GetDevice returns device client.
func (c *Client) GetDevice(ctx context.Context, deviceID string) (*device.Client, error) {
	ctxDev, cancel := context.WithCancel(ctx)
	defer cancel()
	handler := newDeviceHandler(deviceID, cancel)
	resource.DiscoverDevices(ctxDev, c.conn, c.newDiscoveryHandler(handler), coap.WithDeviceID(deviceID))
	cl := handler.Client()
	if cl != nil {
		return cl, nil
	}
	return nil, fmt.Errorf("cannot get device %v: not found", deviceID)
}
