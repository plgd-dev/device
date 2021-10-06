package core

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/plgd-dev/go-coap/v2/udp/client"
	"github.com/plgd-dev/sdk/v2/pkg/net/coap"
	"github.com/plgd-dev/sdk/v2/schema"
)

// GetDeviceByIP gets the device directly via IP address and multicast listen port 5683.
func (c *Client) GetDeviceByIP(ctx context.Context, ip string) (*Device, error) {
	var discoveryConfiguration DiscoveryConfiguration
	if strings.Contains(ip, ":") && !strings.Contains(ip, "[") {
		ip = "[" + ip + "]"
		discoveryConfiguration.MulticastAddressUDP6 = []string{ip + ":5683"}
	} else {
		discoveryConfiguration.MulticastAddressUDP4 = []string{ip + ":5683"}
	}

	findCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	multicastConn, err := DialDiscoveryAddresses(findCtx, discoveryConfiguration, c.errFunc)
	if err != nil {
		return nil, MakeInvalidArgument(fmt.Errorf("could not get the device via ip %s: %w", ip, err))
	}
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()

	h := newDeviceHandler(c.getDeviceConfiguration(), ANY_DEVICE, cancel)
	// we want to just get "oic.wk.d" resource, because links will be get via unicast to /oic/res
	err = DiscoverDevices(findCtx, multicastConn, h, coap.WithResourceType("oic.wk.d"))
	if err != nil {
		return nil, MakeDataLoss(fmt.Errorf("could not get the device from ip %s: %w", ip, err))
	}
	d := h.Device()
	if d == nil {
		return nil, MakeNotFound(fmt.Errorf("no response from the device with ip %s", ip))
	}

	return d, nil
}

// GetDevice performs a multicast and returns a device object if the device responds.
func (c *Client) GetDeviceByMulticast(ctx context.Context, deviceID string, discoveryConfiguration DiscoveryConfiguration) (*Device, error) {
	findCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	multicastConn, err := DialDiscoveryAddresses(findCtx, discoveryConfiguration, c.errFunc)
	if err != nil {
		return nil, MakeInvalidArgument(fmt.Errorf("could not get the device %s: %w", deviceID, err))
	}
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()

	h := newDeviceHandler(c.getDeviceConfiguration(), deviceID, cancel)
	// we want to just get "oic.wk.d" resource, because links will be get via unicast to /oic/res
	err = DiscoverDevices(findCtx, multicastConn, h, coap.WithResourceType("oic.wk.d"))
	if err != nil {
		return nil, MakeDataLoss(fmt.Errorf("could not get the device %s: %w", deviceID, err))
	}
	d := h.Device()
	if d == nil {
		return nil, MakeInternal(fmt.Errorf("no response from the device %s", deviceID))
	}

	return d, nil
}

const ANY_DEVICE = "anydevice"

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

	if h.device != nil || (deviceID != h.deviceID && h.deviceID != ANY_DEVICE) {
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
