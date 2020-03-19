package core

import (
	"context"
	"fmt"
	"sync"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/sdk/schema"
)

type deviceOwnershipHandler struct {
	deviceID string
	cancel   context.CancelFunc

	isSet     bool
	ownership schema.Doxm
	lock      sync.Mutex
	err       error
}

func newDeviceOwnershipHandler(deviceID string, cancel context.CancelFunc) *deviceOwnershipHandler {
	return &deviceOwnershipHandler{deviceID: deviceID, cancel: cancel}
}

func (h *deviceOwnershipHandler) Handle(ctx context.Context, clientConn *gocoap.ClientConn, ownership schema.Doxm) {
	h.lock.Lock()
	defer h.lock.Unlock()
	defer clientConn.Close()
	if h.isSet || ownership.DeviceID != h.deviceID {
		return
	}
	h.ownership = ownership
	h.isSet = true
	h.cancel()
}

func (h *deviceOwnershipHandler) Error(err error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.err == nil {
		h.err = err
	}
}

func (h *deviceOwnershipHandler) Err() error {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.err
}

// GetOwnership gets device's ownership resource.
func (d *Device) GetOwnership(ctx context.Context) (schema.Doxm, error) {
	ctxOwn, cancel := context.WithCancel(ctx)
	defer cancel()

	multicastConn := DialDiscoveryAddresses(ctx, d.cfg.discoveryConfiguration, d.cfg.errFunc)
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()

	h := newDeviceOwnershipHandler(d.DeviceID(), cancel)
	err := DiscoverDeviceOwnership(ctxOwn, multicastConn, DiscoverAllDevices, h)
	if h.isSet {
		return h.ownership, nil
	}
	if err != nil {
		return schema.Doxm{}, err
	}
	err = h.Err()
	if err != nil {
		return schema.Doxm{}, err
	}

	return schema.Doxm{}, fmt.Errorf("device not found")
}
