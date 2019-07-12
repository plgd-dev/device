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
	return DiscoverDevices(ctx, multicastConn, newDiscoveryHandler(c.tlsConfig, c.retryFunc, c.retrieveTimeout, c.errFunc, handler))
}

func newDiscoveryHandler(
	tlsConfig *TLSConfig,
	retryFunc RetryFunc,
	retrieveTimeout time.Duration,
	errFunc ErrFunc,
	h DeviceHandler,
) *discoveryHandler {
	return &discoveryHandler{tlsConfig: tlsConfig, retryFunc: retryFunc, retrieveTimeout: retrieveTimeout, errFunc: errFunc, handler: h}
}

type discoveryHandler struct {
	tlsConfig       *TLSConfig
	retryFunc       RetryFunc
	retrieveTimeout time.Duration
	errFunc         ErrFunc
	handler         DeviceHandler
}

func (h *discoveryHandler) Handle(ctx context.Context, conn *gocoap.ClientConn, links schema.ResourceLinks) {
	defer conn.Close()

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

	h.handler.Handle(ctx, NewDevice(h.tlsConfig, h.retryFunc, h.retrieveTimeout, h.errFunc, deviceID, link.ResourceTypes, links), links)
}

func (h *discoveryHandler) Error(err error) {
	h.handler.Error(err)
}
