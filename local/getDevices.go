package local

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/plgd-dev/kit/log"
	kitNetCoap "github.com/plgd-dev/kit/net/coap"

	codecOcf "github.com/plgd-dev/kit/codec/ocf"
	kitStrings "github.com/plgd-dev/kit/strings"
	"github.com/plgd-dev/sdk/local/core"
	"github.com/plgd-dev/sdk/schema"
	"github.com/plgd-dev/sdk/schema/cloud"
)

// GetDevices discovers devices in the local mode.
// The deviceResourceType is applied on the client side, because len(deviceResourceType) > 1 does not work with Iotivity 1.3.
func (c *Client) GetDevices(
	ctx context.Context,
	opts ...GetDevicesOption,
) (map[string]DeviceDetails, error) {
	cfg := getDevicesOptions{
		err: func(err error) { log.Error(err) },
		getDetails: func(context.Context, *core.Device, schema.ResourceLinks) (interface{}, error) {
			return nil, nil
		},
	}
	for _, o := range opts {
		cfg = o.applyOnGetDevices(cfg)
	}
	getDetails := func(ctx context.Context, d *core.Device, links schema.ResourceLinks) (interface{}, error) {
		return cfg.getDetails(ctx, d, c.PatchResourceLinksEndpoints(links))
	}

	var m sync.Mutex
	var res []DeviceDetails
	devices := func(d DeviceDetails) {
		m.Lock()
		defer m.Unlock()
		res = append(res, d)
	}

	resOwnerships := make(map[string]schema.Doxm)
	ownerships := func(d schema.Doxm) {
		m.Lock()
		defer m.Unlock()
		resOwnerships[d.DeviceID] = d
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ownershipsHandler := newDiscoveryOwnershipsHandler(ctx, cfg.err, ownerships)
	go c.client.GetOwnerships(ctx, core.DiscoverAllDevices, ownershipsHandler)

	handler := newDiscoveryHandler(ctx, cfg.resourceTypes, cfg.err, devices, getDetails)
	if err := c.client.GetDevices(ctx, handler); err != nil {
		return nil, err
	}

	m.Lock()
	defer m.Unlock()

	return setOwnership(mergeDevices(res), resOwnerships), nil
}

// GetDevicesWithHandler discovers devices using a CoAP multicast request via UDP.
// Device resources can be queried in DeviceHandler using device.Client,
func (c *Client) GetDevicesWithHandler(ctx context.Context, handler core.DeviceHandler) error {
	return c.client.GetDevices(ctx, handler)
}

// DeviceDetails describes a device.
type DeviceDetails struct {
	// ID of the device
	ID string
	// Device basic content(oic.wk.d) of /oic/d resource.
	Device schema.Device
	// Details result of function which can be set via option WithGetDetails(), by default it is nil.
	Details interface{}
	// IsSecured is secured.
	IsSecured bool
	// Ownership describes ownership of the device, for unsecure device it is nil.
	Ownership *schema.Doxm
	// Resources list of the device resources.
	Resources []schema.ResourceLink
	// Resources list of the device endpoints.
	Endpoints []schema.Endpoint
}

func newDiscoveryHandler(
	ctx context.Context,
	typeFilter []string,
	errors func(error),
	devices func(DeviceDetails),
	getDetails GetDetailsFunc,
) *discoveryHandler {
	return &discoveryHandler{typeFilter: typeFilter, errors: errors, devices: devices, getDetails: getDetails}
}

type discoveryHandler struct {
	typeFilter []string
	errors     func(error)
	devices    func(DeviceDetails)
	getDetails GetDetailsFunc
}

func (h *discoveryHandler) Error(err error) { h.errors(err) }

func getCloudConfiguration(ctx context.Context, d *core.Device, links schema.ResourceLinks) (*cloud.Configuration, error) {
	for _, l := range links.GetResourceLinks(cloud.ConfigurationResourceType) {
		var ob cloud.Configuration
		var codec codecOcf.VNDOCFCBORCodec
		err := d.GetResourceWithCodec(ctx, l, codec, &ob)
		if err != nil {
			return nil, err
		}
		return &ob, err
	}
	return nil, fmt.Errorf("not found")
}

func getDeviceDetails(ctx context.Context, d *core.Device, links schema.ResourceLinks, getDetails GetDetailsFunc) (out DeviceDetails, _ error) {
	link, ok := links.GetResourceLink("/oic/d")
	var eps []schema.Endpoint
	if ok {
		eps = link.GetEndpoints()
	}

	var device schema.Device
	err := d.GetResource(ctx, link, &device, kitNetCoap.WithInterface("oic.if.baseline"))
	if err != nil {
		return out, err
	}

	isSecured, err := d.IsSecured(ctx, links)
	if err != nil {
		return out, err
	}

	details, err := getDetails(ctx, d, links)
	if err != nil {
		return DeviceDetails{}, err
	}

	return DeviceDetails{
		ID:        d.DeviceID(),
		Device:    device,
		Details:   details,
		IsSecured: isSecured,
		Resources: links,
		Endpoints: eps,
	}, nil
}

func (h *discoveryHandler) Handle(ctx context.Context, d *core.Device, links schema.ResourceLinks) {
	defer d.Close(ctx)

	deviceTypes := make(kitStrings.Set, len(d.DeviceTypes()))
	deviceTypes.Add(d.DeviceTypes()...)
	if !deviceTypes.HasOneOf(h.typeFilter...) {
		return
	}

	devDetails, err := getDeviceDetails(ctx, d, links, h.getDetails)
	if err != nil {
		h.Error(err)
		return
	}

	h.devices(devDetails)
}

func newDiscoveryOwnershipsHandler(
	ctx context.Context,
	errors func(error),
	ownerships func(schema.Doxm),
) *discoveryOwnershipsHandler {
	return &discoveryOwnershipsHandler{errors: errors, ownerships: ownerships}
}

type discoveryOwnershipsHandler struct {
	errors     func(error)
	ownerships func(schema.Doxm)
}

func (h *discoveryOwnershipsHandler) Handle(ctx context.Context, doxm schema.Doxm) {
	h.ownerships(doxm)
}

func (h *discoveryOwnershipsHandler) Error(err error) { h.errors(err) }

func mergeDevices(list []DeviceDetails) map[string]DeviceDetails {
	m := make(map[string]DeviceDetails, len(list))
	for _, i := range list {
		d, ok := m[i.ID]
		if !ok {
			m[i.ID] = i
		} else {
			d.Endpoints = mergeEndpoints(d.Endpoints, i.Endpoints)
		}
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

func setOwnership(devs map[string]DeviceDetails, owns map[string]schema.Doxm) map[string]DeviceDetails {
	for _, o := range owns {
		v := o
		d, ok := devs[o.DeviceID]
		if ok && d.Ownership == nil {
			d.Ownership = &v
			devs[o.DeviceID] = d
		}
	}
	return devs
}
