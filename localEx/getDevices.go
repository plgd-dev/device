package localEx

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/go-ocf/kit/codec/cbor"

	"github.com/go-ocf/go-coap"

	codecOcf "github.com/go-ocf/kit/codec/ocf"
	kitStrings "github.com/go-ocf/kit/strings"
	ocf "github.com/go-ocf/sdk/local"
	ocfschema "github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/schema/cloud"
)

// GetDevices discovers devices in the local mode.
// The deviceResourceType is applied on the client side, because len(deviceResourceType) > 1 does not work with Iotivity 1.3.
func (c *Client) GetDevices(
	ctx context.Context,

	typeFilter []string,
	errors func(error),
) (map[string]DeviceDetails, error) {
	var m sync.Mutex
	var res []DeviceDetails
	devices := func(d DeviceDetails) {
		m.Lock()
		defer m.Unlock()
		res = append(res, d)
	}

	resOwnerships := make(map[string]ocfschema.Doxm)
	ownerships := func(d ocfschema.Doxm) {
		m.Lock()
		defer m.Unlock()
		resOwnerships[d.DeviceID] = d
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ownershipsHandler := newDiscoveryOwnershipsHandler(ctx, errors, ownerships)
	go c.client.GetOwnerships(ctx, ocf.DiscoverAllDevices, ownershipsHandler)

	handler := newDiscoveryHandler(ctx, typeFilter, errors, devices)
	if err := c.client.GetDevices(ctx, handler); err != nil {
		return nil, err
	}

	m.Lock()
	defer m.Unlock()

	return setOwnership(mergeDevices(res), resOwnerships), nil
}

// GetDevicesWithHandler discovers devices using a CoAP multicast request via UDP.
// Device resources can be queried in DeviceHandler using device.Client,
func (c *Client) GetDevicesWithHandler(ctx context.Context, handler ocf.DeviceHandler) error {
	return c.client.GetDevices(ctx, handler)
}

// GetDeviceDetails discovers devices using a CoAP multicast request via UDP
// and provides their details via callback.
func (c *Client) GetDeviceDetails(
	ctx context.Context,
	typeFilter []string,
	errors func(error),
	devices func(DeviceDetails),
) error {
	handler := newDiscoveryHandler(ctx, typeFilter, errors, devices)

	return c.client.GetDevices(ctx, handler)
}

type DeviceDetails struct {
	ID                 string
	Device             ocfschema.Device
	DeviceRaw          []byte
	IsSecured          bool
	Ownership          *ocfschema.Doxm
	CloudConfiguration *cloud.Configuration
	Resources          []ocfschema.ResourceLink
	Endpoints          []ocfschema.Endpoint
}

func newDiscoveryHandler(
	ctx context.Context,
	typeFilter []string,
	errors func(error),
	devices func(DeviceDetails),
) *discoveryHandler {
	return &discoveryHandler{typeFilter: typeFilter, errors: errors, devices: devices}
}

type discoveryHandler struct {
	typeFilter []string
	errors     func(error)
	devices    func(DeviceDetails)
}

func (h *discoveryHandler) Error(err error) { h.errors(err) }

func getCloudConfiguration(ctx context.Context, d *ocf.Device, links ocfschema.ResourceLinks) (*cloud.Configuration, error) {
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

func getDeviceDetails(ctx context.Context, d *ocf.Device, links ocfschema.ResourceLinks) (out DeviceDetails, _ error) {
	link, ok := links.GetResourceLink("/oic/d")
	var eps []ocfschema.Endpoint
	if ok {
		eps = link.GetEndpoints()
	}
	var deviceRaw []byte
	codec := codecOcf.NoCodec{
		MediaType: uint16(coap.AppOcfCbor),
	}
	err := d.GetResourceWithCodec(ctx, link, codec, &deviceRaw)
	if err != nil {
		return out, err
	}

	var device ocfschema.Device
	err = cbor.Decode(deviceRaw, &device)
	if err != nil {
		return out, err
	}

	isSecured, err := d.IsSecured(ctx, links)
	if err != nil {
		return out, err
	}
	cloudCfg, _ := getCloudConfiguration(ctx, d, links)

	return DeviceDetails{
		ID:                 d.DeviceID(),
		Device:             device,
		DeviceRaw:          deviceRaw,
		IsSecured:          isSecured,
		CloudConfiguration: cloudCfg,
		Resources:          links,
		Endpoints:          eps,
	}, nil
}

func (h *discoveryHandler) Handle(ctx context.Context, d *ocf.Device, links ocfschema.ResourceLinks) {
	defer d.Close(ctx)

	deviceTypes := make(kitStrings.Set, len(d.DeviceTypes()))
	deviceTypes.Add(d.DeviceTypes()...)
	if !deviceTypes.HasOneOf(h.typeFilter...) {
		return
	}

	devDetails, err := getDeviceDetails(ctx, d, links)
	if err != nil {
		h.Error(err)
		return
	}

	h.devices(devDetails)
}

func newDiscoveryOwnershipsHandler(
	ctx context.Context,
	errors func(error),
	ownerships func(ocfschema.Doxm),
) *discoveryOwnershipsHandler {
	return &discoveryOwnershipsHandler{errors: errors, ownerships: ownerships}
}

type discoveryOwnershipsHandler struct {
	errors     func(error)
	ownerships func(ocfschema.Doxm)
}

func (h *discoveryOwnershipsHandler) Handle(ctx context.Context, doxm ocfschema.Doxm) {
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

func mergeEndpoints(a, b []ocfschema.Endpoint) []ocfschema.Endpoint {
	eps := make([]ocfschema.Endpoint, 0, len(a)+len(b))
	eps = append(eps, a...)
	eps = append(eps, b...)
	sort.SliceStable(eps, func(i, j int) bool { return eps[i].URI < eps[j].URI })
	sort.SliceStable(eps, func(i, j int) bool { return eps[i].Priority < eps[j].Priority })
	out := make([]ocfschema.Endpoint, 0, len(eps))
	var last string
	for _, e := range eps {
		if last != e.URI {
			out = append(out, e)
		}
		last = e.URI
	}
	return out
}

func setOwnership(devs map[string]DeviceDetails, owns map[string]ocfschema.Doxm) map[string]DeviceDetails {
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
