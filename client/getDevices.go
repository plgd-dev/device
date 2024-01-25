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

package client

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	kitStrings "github.com/plgd-dev/kit/v2/strings"
)

func getDetails(ctx context.Context, d *core.Device, links schema.ResourceLinks, optsArgs ...func(message.Options) message.Options) (interface{}, error) {
	link := links.GetResourceLinks(device.ResourceType)
	if len(link) == 0 {
		return nil, fmt.Errorf("cannot find device resource at links %+v", links)
	}
	var dev device.Device
	opts := []func(message.Options) message.Options{
		coap.WithInterface(interfaces.OC_IF_BASELINE),
	}
	opts = append(opts, optsArgs...)
	err := d.GetResource(ctx, link[0], &dev, opts...)
	if err != nil {
		return nil, err
	}
	return &dev, nil
}

type ownership struct {
	doxm   *doxm.Doxm
	status OwnershipStatus
}

func (c *Client) getDevicesAppendDeviceByIP(ctx context.Context, ip string, resourceTypes []string, getDetails func(ctx context.Context, d *core.Device, links schema.ResourceLinks, optsArgs ...func(message.Options) message.Options) (interface{}, error), appendDevice func(d DeviceDetails)) {
	d, err := c.GetDeviceDetailsByIP(ctx, ip, WithGetDetails(getDetails))
	if err != nil {
		return
	}
	devLinks := d.Resources.GetResourceLinks(device.ResourceType)
	if len(devLinks) == 0 {
		return
	}
	if len(resourceTypes) == 0 {
		appendDevice(d)
		return
	}
	for _, t := range resourceTypes {
		if kitStrings.SliceContains(devLinks[0].ResourceTypes, t) {
			appendDevice(d)
			return
		}
	}
}

// GetDevicesDetails gets devices by multicast and devices found by ip. Each device are stored or refreshed in cache. When the device expiration time has expired,
// the device will be removed from cache. The device expiration time is prolonged by using the device.
func (c *Client) GetDevicesDetails(
	ctx context.Context,
	opts ...GetDevicesOption,
) (map[string]DeviceDetails, error) {
	cfg := getDevicesOptions{
		getDetails:             getDetails,
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
		useDeviceID:            c.useDeviceIDInQuery,
	}
	for _, o := range opts {
		cfg = o.applyOnGetDevices(cfg)
	}
	var m sync.Mutex
	resOwnerships := make(map[string]ownership)
	ownerships := func(deviceID string, d ownership) {
		m.Lock()
		defer m.Unlock()
		resOwnerships[deviceID] = d
	}

	getDetails := func(ctx context.Context, d *core.Device, links schema.ResourceLinks, optsArgs ...func(message.Options) message.Options) (interface{}, error) {
		links = patchResourceLinksEndpoints(links, c.disableUDPEndpoints)
		details, err := cfg.getDetails(ctx, d, links, optsArgs...)
		if err == nil && d.IsSecured() {
			doxm, ownErr := d.GetOwnership(ctx, links, optsArgs...)
			if ownErr == nil {
				ownerships(d.DeviceID(), ownership{
					doxm:   &doxm,
					status: OwnershipStatus_Unknown, // will be resolved later
				})
			} else if isDeviceOwnedByOther(ownErr) {
				ownerships(d.DeviceID(), ownership{
					status: OwnershipStatus_OwnedByOther,
				})
			}
		}
		return details, err
	}

	var res []DeviceDetails
	devices := func(d DeviceDetails) {
		m.Lock()
		defer m.Unlock()
		res = append(res, d)
	}

	devicesFoundByIP := c.GetAllDeviceIDsFoundByIP()
	var wg sync.WaitGroup
	wg.Add(len(devicesFoundByIP))
	for _, ip := range devicesFoundByIP {
		go func(ip string) {
			defer wg.Done()
			c.getDevicesAppendDeviceByIP(ctx, ip, cfg.resourceTypes, getDetails, devices)
		}(ip)
	}

	handler := newDiscoveryHandler(cfg.resourceTypes, c.logger, devices, getDetails, c.deviceCache, c.disableUDPEndpoints, cfg.useDeviceID)
	if err := c.client.GetDevicesByMulticast(ctx, cfg.discoveryConfiguration, handler); err != nil {
		return nil, err
	}

	wg.Wait()
	m.Lock()
	defer m.Unlock()
	ownerID, _ := c.client.GetSdkOwnerID()
	return setOwnership(ownerID, mergeDevices(res), resOwnerships), nil
}

// GetDevicesWithHandler discovers devices using a CoAP multicast request via UDP.
// Device resources can be queried in DeviceHandler using device.Client,
func (c *Client) GetDevicesWithHandler(ctx context.Context, handler core.DeviceMulticastHandler, opts ...GetDevicesWithHandlerOption) error {
	cfg := getDevicesWithHandlerOptions{
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnGetGetDevicesWithHandler(cfg)
	}
	return c.client.GetDevicesByMulticast(ctx, cfg.discoveryConfiguration, handler)
}

// OwnershipStatus describes ownership status of the device
type OwnershipStatus string

const (
	// OwnershipStatus_ReadyToBeOwned the device is ready to be owned.
	OwnershipStatus_ReadyToBeOwned OwnershipStatus = "readytobeowned"
	// OwnershipStatus_Owned the device is owned.
	OwnershipStatus_Owned OwnershipStatus = "owned"
	// OwnershipStatus_OwnedByOther the device is owned by another user.
	OwnershipStatus_OwnedByOther OwnershipStatus = "ownedbyother"
	// OwnershipStatus_Unknown the device is unsecure or cannot obtain his status.
	OwnershipStatus_Unknown OwnershipStatus = "unknown"
)

// DeviceDetails describes a device.
type DeviceDetails struct {
	// ID of the device
	ID string
	// IP used to find this device
	FoundByIP string
	// Details result of function which can be set via option WithGetDetails(), by default it is nil.
	Details interface{}
	// IsSecured is secured.
	IsSecured bool
	// Ownership describes ownership of the device, for unsecure device it is nil.
	Ownership *doxm.Doxm
	// Resources list of the device resources.
	Resources schema.ResourceLinks
	// Resources list of the device endpoints.
	Endpoints []schema.Endpoint
	// Ownership status
	OwnershipStatus OwnershipStatus
}

func newDiscoveryHandler(
	typeFilter []string,
	logger core.Logger,
	devices func(DeviceDetails),
	getDetails GetDetailsFunc,
	deviceCache *DeviceCache,
	disableUDPEndpoints bool,
	useDeviceID bool,
) *discoveryHandler {
	return &discoveryHandler{typeFilter: typeFilter, logger: logger, devices: devices, getDetails: getDetails, deviceCache: deviceCache, disableUDPEndpoints: disableUDPEndpoints, useDeviceID: useDeviceID}
}

type detailsWasSet struct {
	sync.Mutex
	wasSet bool
}

type discoveryHandler struct {
	typeFilter          []string
	logger              core.Logger
	devices             func(DeviceDetails)
	getDetails          GetDetailsFunc
	deviceCache         *DeviceCache
	disableUDPEndpoints bool
	useDeviceID         bool

	getDetailsWasCalled sync.Map
}

func (h *discoveryHandler) Error(err error) { h.logger.Debug(err.Error()) }

func getDeviceDetails(ctx context.Context, dev *core.Device, links schema.ResourceLinks, getDetails GetDetailsFunc, opts []func(message.Options) message.Options) (out DeviceDetails, _ error) {
	link, ok := links.GetResourceLink(device.ResourceURI)
	var eps []schema.Endpoint
	if ok {
		eps = link.GetEndpoints()
	}

	isSecured := dev.IsSecured()
	var details interface{}
	if getDetails != nil {
		d, err := getDetails(ctx, dev, links, opts...)
		if err != nil {
			return DeviceDetails{}, err
		}
		details = d
	}

	return DeviceDetails{
		ID:              dev.DeviceID(),
		FoundByIP:       dev.FoundByIP(),
		Details:         details,
		IsSecured:       isSecured,
		Resources:       links,
		Endpoints:       eps,
		OwnershipStatus: OwnershipStatus_Unknown,
	}, nil
}

func (h *discoveryHandler) Handle(ctx context.Context, newdev *core.Device) {
	dev, _ := h.deviceCache.UpdateOrStoreDeviceWithExpiration(newdev)
	links, err := getLinksDevice(ctx, dev, h.disableUDPEndpoints)
	if err != nil {
		dev2, ok := h.deviceCache.LoadAndDeleteDevice(dev.DeviceID())
		if ok {
			if errC := dev2.Close(ctx); errC != nil {
				h.logger.Debugf("discovery error: %w", errC)
			}
		}
		return
	}
	deviceTypes := make(kitStrings.Set, len(dev.DeviceTypes()))
	deviceTypes.Add(dev.DeviceTypes()...)
	if !deviceTypes.HasOneOf(h.typeFilter...) {
		return
	}

	devDetails, err := h.getDeviceDetails(ctx, dev, links)
	if err != nil {
		h.Error(err)
		return
	}

	h.devices(devDetails)
}

func (h *discoveryHandler) getDeviceDetails(ctx context.Context, d *core.Device, links schema.ResourceLinks) (out DeviceDetails, _ error) {
	getDetails := h.getDetails
	v, _ := h.getDetailsWasCalled.LoadOrStore(d.DeviceID(), &detailsWasSet{})
	m := v.(*detailsWasSet) //nolint:forcetypeassert
	m.Lock()
	defer m.Unlock()
	if m.wasSet {
		getDetails = nil
	}
	opts := make([]func(message.Options) message.Options, 0, 1)
	if h.useDeviceID {
		opts = append(opts, coap.WithDeviceID(d.DeviceID()))
	}

	devDetails, err := getDeviceDetails(ctx, d, links, getDetails, opts)
	if err == nil {
		m.wasSet = true
	}
	return devDetails, err
}

func mergeDevices(list []DeviceDetails) map[string]DeviceDetails {
	m := make(map[string]DeviceDetails, len(list))
	for _, i := range list {
		d, ok := m[i.ID]
		if !ok {
			m[i.ID] = i
			d = i
		} else {
			d.Endpoints = mergeEndpoints(d.Endpoints, i.Endpoints)
		}
		if i.Details != nil {
			d.Details = i.Details
		}
		m[i.ID] = d
	}
	return m
}

func mergeEndpoints(a, b []schema.Endpoint) []schema.Endpoint {
	eps := make([]schema.Endpoint, 0, len(a)+len(b))
	eps = append(eps, a...)
	eps = append(eps, b...)
	sort.SliceStable(eps, func(i, j int) bool { return eps[i].URI < eps[j].URI })
	sort.SliceStable(eps, func(i, j int) bool { return eps[i].Priority < eps[j].Priority })
	out := make([]schema.Endpoint, 0, len(eps))
	var last string
	for _, e := range eps {
		if last != e.URI {
			out = append(out, e)
		}
		last = e.URI
	}
	return out
}

func setOwnership(ownerID string, devs map[string]DeviceDetails, owns map[string]ownership) map[string]DeviceDetails {
	for deviceID, o := range owns {
		d, ok := devs[deviceID]
		if ok && d.Ownership == nil {
			if o.doxm == nil {
				d.OwnershipStatus = o.status
			} else {
				d.Ownership = o.doxm
				switch o.doxm.OwnerID {
				case uuid.Nil.String():
					d.OwnershipStatus = OwnershipStatus_ReadyToBeOwned
				case ownerID:
					d.OwnershipStatus = OwnershipStatus_Owned
				default:
					d.OwnershipStatus = OwnershipStatus_OwnedByOther
				}
			}
			devs[deviceID] = d
		}
	}
	return devs
}
