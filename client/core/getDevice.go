// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package core

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/go-coap/v3/udp/client"
)

// GetDeviceByIP gets the device directly via IP address and multicast listen port 5683.
func (c *Client) GetDeviceByIP(ctx context.Context, ip string) (*Device, error) {
	devices, err := c.GetDevicesByIP(ctx, ip)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, MakeNotFound(fmt.Errorf("no response from the device with ip %s", ip))
	}
	for _, d := range devices {
		err = d.Close(ctx)
		if err != nil {
			c.logger.Debugf("get device by ip error: %s", err.Error())
		}
	}
	return devices[0], nil
}

func getAddress(ip string) (addr string, isIpv4 bool, err error) {
	if strings.Contains(ip, ".") {
		host, port, err := net.SplitHostPort(ip)
		if err != nil {
			host, port, err = net.SplitHostPort(ip + ":5683")
		}
		if err != nil {
			return "", false, err
		}
		return host + ":" + port, true, nil
	}
	if !strings.Contains(ip, "[") {
		ip = "[" + ip + "]:5683"
	}
	host, port, err := net.SplitHostPort(ip)
	if err != nil {
		host, port, err = net.SplitHostPort(ip + ":5683")
	}
	if err != nil {
		return "", false, err
	}
	return "[" + host + "]:" + port, true, nil
}

// GetDevicesByIP gets the devices directly via IP address and multicast listen port 5683.
func (c *Client) GetDevicesByIP(ctx context.Context, ip string) ([]*Device, error) {
	var discoveryConfiguration DiscoveryConfiguration

	addr, isIPv4, err := getAddress(ip)
	if err != nil {
		return nil, MakeInvalidArgument(fmt.Errorf("could not get the device via ip %s: %w", ip, err))
	}
	if isIPv4 {
		discoveryConfiguration.MulticastAddressUDP4 = []string{addr}
	} else {
		discoveryConfiguration.MulticastAddressUDP6 = []string{addr}
	}

	findCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	multicastConn, err := DialDiscoveryAddresses(findCtx, discoveryConfiguration, func(err error) { c.logger.Debug(err.Error()) })
	if err != nil {
		return nil, MakeInvalidArgument(fmt.Errorf("could not get the device via ip %s: %w", ip, err))
	}
	defer func() {
		for _, conn := range multicastConn {
			if errC := conn.Close(); errC != nil {
				c.logger.Debug(fmt.Errorf("get device by ip error: cannot close connection(%s): %w", conn.mcastaddr, errC).Error())
			}
		}
	}()

	h := newDevicesHandler(c.getDeviceConfiguration(), ANY_DEVICE, cancel)
	// we want to just get "oic.wk.d" resource, because links will be get via unicast to /oic/res
	err = DiscoverDevices(findCtx, multicastConn, h, coap.WithResourceType(device.ResourceType))
	if err != nil {
		return nil, MakeDataLoss(fmt.Errorf("could not get the devices from ip %s: %w", ip, err))
	}
	devices := h.Devices()
	if len(devices) == 0 {
		return nil, MakeNotFound(fmt.Errorf("no response from the devices with ip %s", ip))
	}
	for _, d := range devices {
		d.setFoundByIP(ip)
	}
	return devices, nil
}

// GetDeviceByMulticast performs a multicast and returns a device object if the device responds.
func (c *Client) GetDeviceByMulticast(ctx context.Context, deviceID string, discoveryConfiguration DiscoveryConfiguration) (*Device, error) {
	findCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	multicastConn, err := DialDiscoveryAddresses(findCtx, discoveryConfiguration, func(err error) { c.logger.Debug(err.Error()) })
	if err != nil {
		return nil, MakeInvalidArgument(fmt.Errorf("could not get the device %s: %w", deviceID, err))
	}
	defer func() {
		for _, conn := range multicastConn {
			if errC := conn.Close(); errC != nil {
				c.logger.Debug(fmt.Errorf("get device by multicast error: cannot close connection(%s): %w", conn.mcastaddr, errC).Error())
			}
		}
	}()

	h := newDevicesHandler(c.getDeviceConfiguration(), deviceID, cancel)
	// we want to just get "oic.wk.d" resource, because links will be get via unicast to /oic/res
	err = DiscoverDevices(findCtx, multicastConn, h, coap.WithResourceType(device.ResourceType), coap.WithDeviceID(deviceID))
	if err != nil {
		return nil, MakeDataLoss(fmt.Errorf("could not get the device %s: %w", deviceID, err))
	}
	d := h.Devices()
	if len(d) == 0 {
		err = h.Err()
		if err != nil {
			return nil, MakeInternal(fmt.Errorf("no response from the device %s: %w", deviceID, err))
		}
		return nil, MakeInternal(fmt.Errorf("no response from the device %s", deviceID))
	}

	return d[0], nil
}

const ANY_DEVICE = "anydevice"

func newDevicesHandler(
	deviceCfg DeviceConfiguration,
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
	deviceCfg DeviceConfiguration
	deviceID  string
	cancel    context.CancelFunc

	lock    sync.Mutex
	devices []*Device
	err     error
}

func (h *deviceHandler) Devices() []*Device {
	h.lock.Lock()
	defer h.lock.Unlock()
	devices := h.devices
	h.devices = nil
	return devices
}

func (h *deviceHandler) Handle(_ context.Context, conn *client.Conn, links schema.ResourceLinks) {
	if errC := conn.Close(); errC != nil {
		h.deviceCfg.Logger.Debug(fmt.Errorf("device handler cannot close connection: %w", errC).Error())
	}
	h.lock.Lock()
	defer h.lock.Unlock()
	links = links.GetResourceLinks(device.ResourceType)
	if len(links) == 0 {
		h.err = MakeUnavailable(fmt.Errorf("cannot get %v resourceType for device: not found", device.ResourceType))
		return
	}
	if len(h.devices) > 0 {
		return
	}
	var errs *multierror.Error
	for _, link := range links {
		deviceID := link.GetDeviceID()
		if deviceID == "" {
			errs = multierror.Append(errs, MakeUnavailable(fmt.Errorf("cannot determine deviceID")))
			continue
		}
		if deviceID != h.deviceID && h.deviceID != ANY_DEVICE {
			continue
		}
		if len(link.ResourceTypes) == 0 {
			errs = multierror.Append(errs, MakeUnavailable(fmt.Errorf("cannot get resource types for %v: is empty", deviceID)))
			continue
		}
		h.devices = append(h.devices, NewDevice(h.deviceCfg, deviceID, link.ResourceTypes, link.GetEndpoints))
	}
	h.err = errs.ErrorOrNil()
	if len(h.devices) > 0 {
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
