package client

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	kitStrings "github.com/plgd-dev/kit/v2/strings"
)

func getDetails(ctx context.Context, d *core.Device, links schema.ResourceLinks) (interface{}, error) {
	link := links.GetResourceLinks(device.ResourceType)
	if len(link) == 0 {
		return nil, fmt.Errorf("cannot find device resource at links %+v", links)
	}
	var dev device.Device
	err := d.GetResource(ctx, link[0], &dev, coap.WithInterface(interfaces.OC_IF_BASELINE))
	if err != nil {
		return nil, err
	}
	return &dev, nil
}

type ownership struct {
	doxm   *doxm.Doxm
	status OwnershipStatus
}

// GetDevices discovers devices in the local mode.
// The deviceResourceType is applied on the client side, because len(deviceResourceType) > 1 does not work with Iotivity 1.3.
func (c *Client) GetDevices(
	ctx context.Context,
	opts ...GetDevicesOption,
) (map[string]DeviceDetails, error) {
	cfg := getDevicesOptions{
		err:                    c.errors,
		getDetails:             getDetails,
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
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

	getDetails := func(ctx context.Context, d *core.Device, links schema.ResourceLinks) (interface{}, error) {
		links = patchResourceLinksEndpoints(links, c.disableUDPEndpoints)
		details, err := cfg.getDetails(ctx, d, links)
		if err == nil && d.IsSecured() {
			doxm, ownErr := d.GetOwnership(ctx, links)
			if ownErr == nil {
				ownerships(d.DeviceID(), ownership{
					doxm:   &doxm,
					status: OwnershipStatus_Unknown, // will be resolved later
				})
			} else if strings.Contains(ownErr.Error(), "x509: certificate signed by unknown authority") {
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

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	handler := newDiscoveryHandler(cfg.resourceTypes, cfg.err, devices, getDetails, c.deviceCache, c.disableUDPEndpoints)
	if err := c.client.GetDevicesV2(ctx, cfg.discoveryConfiguration, handler); err != nil {
		return nil, err
	}

	m.Lock()
	defer m.Unlock()
	ownerID, _ := c.client.GetSdkOwnerID()
	return setOwnership(ownerID, mergeDevices(res), resOwnerships), nil
}

// GetDevicesWithHandler discovers devices using a CoAP multicast request via UDP.
// Device resources can be queried in DeviceHandler using device.Client,
func (c *Client) GetDevicesWithHandler(ctx context.Context, handler core.DeviceHandlerV2, opts ...GetDevicesWithHandlerOption) error {
	cfg := getDevicesWithHandlerOptions{
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnGetGetDevicesWithHandler(cfg)
	}
	return c.client.GetDevicesV2(ctx, cfg.discoveryConfiguration, handler)
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
	// Details result of function which can be set via option WithGetDetails(), by default it is nil.
	Details interface{}
	// IsSecured is secured.
	IsSecured bool
	// Ownership describes ownership of the device, for unsecure device it is nil.
	Ownership *doxm.Doxm
	// Resources list of the device resources.
	Resources []schema.ResourceLink
	// Resources list of the device endpoints.
	Endpoints []schema.Endpoint
	// Ownership status
	OwnershipStatus OwnershipStatus
}

func newDiscoveryHandler(
	typeFilter []string,
	errors func(error),
	devices func(DeviceDetails),
	getDetails GetDetailsFunc,
	deviceCache *refDeviceCache,
	disableUDPEndpoints bool,
) *discoveryHandler {
	return &discoveryHandler{typeFilter: typeFilter, errors: errors, devices: devices, getDetails: getDetails, deviceCache: deviceCache, disableUDPEndpoints: disableUDPEndpoints}
}

type detailsWasSet struct {
	sync.Mutex
	wasSet bool
}

type discoveryHandler struct {
	typeFilter          []string
	errors              func(error)
	devices             func(DeviceDetails)
	getDetails          GetDetailsFunc
	deviceCache         *refDeviceCache
	disableUDPEndpoints bool

	getDetailsWasCalled sync.Map
}

func (h *discoveryHandler) Error(err error) { h.errors(err) }

func getDeviceDetails(ctx context.Context, dev *core.Device, links schema.ResourceLinks, getDetails GetDetailsFunc) (out DeviceDetails, _ error) {
	link, ok := links.GetResourceLink(device.ResourceURI)
	var eps []schema.Endpoint
	if ok {
		eps = link.GetEndpoints()
	}

	isSecured := dev.IsSecured()
	var details interface{}
	if getDetails != nil {
		d, err := getDetails(ctx, dev, links)
		if err != nil {
			return DeviceDetails{}, err
		}
		details = d
	}

	return DeviceDetails{
		ID:              dev.DeviceID(),
		Details:         details,
		IsSecured:       isSecured,
		Resources:       links,
		Endpoints:       eps,
		OwnershipStatus: OwnershipStatus_Unknown,
	}, nil
}

func (h *discoveryHandler) Handle(ctx context.Context, d *core.Device) {
	newRefDev := NewRefDevice(d)
	refDev, stored := h.deviceCache.TryStoreDeviceToTemporaryCache(newRefDev)
	defer refDev.Release(ctx)
	links, err := getLinksRefDevice(ctx, refDev, h.disableUDPEndpoints)
	d = refDev.Device()
	if err != nil {
		refDev.Device().Close(ctx)
		h.deviceCache.RemoveDevice(ctx, refDev.DeviceID(), refDev)
		if stored {
			return
		}
		refDev, stored = h.deviceCache.TryStoreDeviceToTemporaryCache(newRefDev)
		if !stored {
			newRefDev.Release(ctx)
			return
		}
		links, err = getLinksRefDevice(ctx, refDev, h.disableUDPEndpoints)
		if err != nil {
			refDev.Device().Close(ctx)
			h.deviceCache.RemoveDevice(ctx, refDev.DeviceID(), refDev)
			refDev.Release(ctx)
			return
		}
		d = refDev.Device()
	} else if !stored {
		newRefDev.Release(ctx)
	}

	deviceTypes := make(kitStrings.Set, len(d.DeviceTypes()))
	deviceTypes.Add(d.DeviceTypes()...)
	if !deviceTypes.HasOneOf(h.typeFilter...) {
		return
	}

	devDetails, err := h.getDeviceDetails(ctx, d, links)
	if err != nil {
		h.Error(err)
		return
	}

	h.devices(devDetails)
}

func (h *discoveryHandler) getDeviceDetails(ctx context.Context, d *core.Device, links schema.ResourceLinks) (out DeviceDetails, _ error) {
	getDetails := h.getDetails
	v, _ := h.getDetailsWasCalled.LoadOrStore(d.DeviceID(), &detailsWasSet{})
	m := v.(*detailsWasSet)
	m.Lock()
	defer m.Unlock()
	if m.wasSet {
		getDetails = nil
	}
	devDetails, err := getDeviceDetails(ctx, d, links, getDetails)
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
