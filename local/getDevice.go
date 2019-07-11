package local

import (
	"context"
	"fmt"
	"sync"
	"time"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/schema"
)

// GetDevice performs a multicast and returns a device object if the device responds.
func (c *Client) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	multicastConn := DialDiscoveryAddresses(ctx, c.errFunc)
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()

	h := newDeviceHandler(deviceID, c.tlsConfig, c.retryFunc, c.retrieveTimeout, c.errFunc, cancel)
	err := DiscoverDevices(ctx, multicastConn, h)
	if err != nil {
		return nil, fmt.Errorf("could not get the device %s: %v", deviceID, err)
	}
	d := h.Device()
	if d == nil {
		return nil, fmt.Errorf("no response from the device %s", deviceID)
	}
	return d, nil
}

func newDeviceHandler(
	deviceID string,
	tlsConfig *TLSConfig,
	retryFunc RetryFunc,
	retrieveTimeout time.Duration,
	errFunc ErrFunc,
	cancel context.CancelFunc,
) *deviceHandler {
	return &deviceHandler{
		deviceID:        deviceID,
		tlsConfig:       tlsConfig,
		retryFunc:       retryFunc,
		retrieveTimeout: retrieveTimeout,
		errFunc:         errFunc,
		cancel:          cancel,
	}
}

type deviceHandler struct {
	deviceID        string
	tlsConfig       *TLSConfig
	retryFunc       RetryFunc
	retrieveTimeout time.Duration
	errFunc         ErrFunc
	cancel          context.CancelFunc

	lock   sync.Mutex
	device *Device
	err    error
}

func (h *deviceHandler) Device() *Device {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.device
}

func (h *deviceHandler) Handle(ctx context.Context, conn *gocoap.ClientConn, links schema.ResourceLinks) {
	h.lock.Lock()
	defer h.lock.Unlock()

	link, ok := links.GetResourceLink("/oic/d")
	if !ok {
		h.err = fmt.Errorf("cannot get link to /oic/d")
		return
	}
	deviceID := link.GetDeviceID()

	if h.device != nil || deviceID != h.deviceID {
		return
	}
	defer h.cancel()
	client := coap.NewClient(conn)
	defer client.Close()

	h.device = NewDevice(h.tlsConfig, h.retryFunc, h.retrieveTimeout, h.errFunc, deviceID, links)
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
