package local

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-ocf/kit/net/coap"

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

	h := newDeviceHandler(deviceID, c.tlsConfig, c.retryFuncFactory, c.retrieveTimeout, c.errFunc, c.resolveEndpointsFunc, c.dialOptions, c.discoveryConfiguration, cancel)
	err := DiscoverDevices(ctx, multicastConn, h)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get the device %s: %v", deviceID, err)
	}
	d, dlinks := h.Device()
	if d == nil {
		return nil, nil, fmt.Errorf("no response from the device %s", deviceID)
	}
	return d, dlinks, nil
}

func newDeviceHandler(
	deviceID string,
	tlsConfig *TLSConfig,
	retryFuncFactory RetryFuncFactory,
	retrieveTimeout time.Duration,
	errFunc ErrFunc,
	resolveEndpointsFunc ResolveEndpointsFunc,
	dialOptions []coap.DialOptionFunc,
	discoveryConfiguration DiscoveryConfiguration,
	cancel context.CancelFunc,
) *deviceHandler {
	return &deviceHandler{
		deviceID:               deviceID,
		tlsConfig:              tlsConfig,
		retryFuncFactory:       retryFuncFactory,
		retrieveTimeout:        retrieveTimeout,
		errFunc:                errFunc,
		resolveEndpointsFunc:   resolveEndpointsFunc,
		dialOptions:            dialOptions,
		discoveryConfiguration: discoveryConfiguration,
		cancel:                 cancel,
	}
}

type deviceHandler struct {
	deviceID               string
	tlsConfig              *TLSConfig
	retryFuncFactory       RetryFuncFactory
	retrieveTimeout        time.Duration
	errFunc                ErrFunc
	resolveEndpointsFunc   ResolveEndpointsFunc
	dialOptions            []coap.DialOptionFunc
	cancel                 context.CancelFunc
	discoveryConfiguration DiscoveryConfiguration

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

	link, ok := links.GetResourceLink("/oic/d")
	if !ok {
		h.err = fmt.Errorf("cannot get link to /oic/d")
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
	_, err := h.resolveEndpointsFunc(ctx, "/oic/d", links)
	if err != nil {
		h.err = fmt.Errorf("cannot resolve endpoints for href %v  of %v : %v ", link.Href, deviceID, err)
		return
	}
	d := NewDevice(h.tlsConfig, h.retryFuncFactory, h.retrieveTimeout, h.errFunc, h.resolveEndpointsFunc, h.dialOptions, h.discoveryConfiguration, deviceID, link.ResourceTypes, links)

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
