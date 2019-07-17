package local

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ocf/sdk/schema"

	gocoap "github.com/go-ocf/go-coap"
)

// DeviceHandler conveys device connections and errors during discovery.
type DeviceHandler interface {
	// Handle gets a device connection and is responsible for closing it.
	Handle(ctx context.Context, device *Device, deviceLinks schema.ResourceLinks)
	// Error gets errors during discovery.
	Error(err error)
}

// GetDevices discovers devices using a CoAP multicast request via UDP.
// Device resources can be queried in DeviceHandler using device.Client,
func (c *Client) GetDevices(ctx context.Context, handler DeviceHandler) error {
	multicastConn := DialDiscoveryAddresses(ctx, c.errFunc)
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()
	return DiscoverDevices(ctx, multicastConn, newDiscoveryHandler(c.tlsConfig, c.retryFunc, c.retrieveTimeout, c.errFunc, c.resolveEndpointsFunc, handler))
}

func newDiscoveryHandler(
	tlsConfig *TLSConfig,
	retryFunc RetryFunc,
	retrieveTimeout time.Duration,
	errFunc ErrFunc,
	resolveEndpointsFunc ResolveEndpointsFunc,
	h DeviceHandler,
) *discoveryHandler {
	return &discoveryHandler{tlsConfig: tlsConfig, retryFunc: retryFunc, retrieveTimeout: retrieveTimeout, errFunc: errFunc, resolveEndpointsFunc: resolveEndpointsFunc, handler: h}
}

type discoveryHandler struct {
	tlsConfig            *TLSConfig
	retryFunc            RetryFunc
	retrieveTimeout      time.Duration
	errFunc              ErrFunc
	resolveEndpointsFunc ResolveEndpointsFunc
	handler              DeviceHandler
}

func (h *discoveryHandler) Handle(ctx context.Context, conn *gocoap.ClientConn, links schema.ResourceLinks) {
	conn.Close()

	link, ok := links.GetResourceLink("/oic/d")
	if !ok {
		h.handler.Error(fmt.Errorf("cannot get link to /oic/d"))
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
	endpoints, err := h.resolveEndpointsFunc(ctx, "/oic/d", links)
	if err != nil {
		h.handler.Error(fmt.Errorf("cannot resolve endpoints for href %v  of %v : %v ", link.Href, deviceID, err))
		return
	}

	d := NewDevice(h.tlsConfig, h.retryFunc, h.retrieveTimeout, h.errFunc, h.resolveEndpointsFunc, deviceID, link.ResourceTypes, links)
	_, err = d.connectToEndpoints(ctx, endpoints)
	if err != nil {
		d.Close(ctx)
		h.handler.Error(fmt.Errorf("cannot connect to /oic/d for %v: %v", deviceID, err))
		return
	}

	h.handler.Handle(ctx, d, links)
}

func (h *discoveryHandler) Error(err error) {
	h.handler.Error(err)
}
