package client

import (
	"context"

	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/client/core/otm"
	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/doxm"
)

// TODO: remove whole file in v2

// Deprecated: use Device
type RefDevice struct {
	dev *core.Device
}

func NewRefDevice(dev *core.Device) *RefDevice {
	return &RefDevice{dev: dev}
}

func (d *RefDevice) Acquire() {
	// backward compatibility
}

func (d *RefDevice) Release(ctx context.Context) error {
	// backward compatibility
	return nil
}

func (d *RefDevice) DeviceID() string {
	return d.Device().DeviceID()
}

func (d *RefDevice) Device() *core.Device {
	return d.dev
}

func (d *RefDevice) GetDeviceDetails(ctx context.Context, links schema.ResourceLinks, getDetails GetDetailsFunc) (out DeviceDetails, _ error) {
	return getDeviceDetails(ctx, d.Device(), links, getDetails)
}

func (d *RefDevice) GetResourceWithCodec(
	ctx context.Context,
	link schema.ResourceLink,
	codec coap.Codec,
	response interface{},
	options ...coap.OptionFunc,
) error {
	return d.Device().GetResourceWithCodec(ctx, link, codec, response, options...)
}

func (d *RefDevice) ObserveResourceWithCodec(
	ctx context.Context,
	link schema.ResourceLink,
	codec coap.Codec,
	handler core.ObservationHandler,
	options ...coap.OptionFunc,
) (observationID string, _ error) {
	return d.Device().ObserveResourceWithCodec(ctx, link, codec, handler, options...)
}

func (d *RefDevice) ObserveResource(
	ctx context.Context,
	link schema.ResourceLink,
	handler core.ObservationHandler,
	options ...coap.OptionFunc,
) (observationID string, _ error) {
	return d.Device().ObserveResource(ctx, link, handler, options...)
}

func (d *RefDevice) StopObservingResource(
	ctx context.Context,
	observationID string,
) (bool, error) {
	return d.Device().StopObservingResource(ctx, observationID)
}

func (d *RefDevice) IsSecured() bool {
	return d.Device().IsSecured()
}

func (d *RefDevice) UpdateResource(
	ctx context.Context,
	link schema.ResourceLink,
	request interface{},
	response interface{},
	options ...coap.OptionFunc,
) error {
	return d.Device().UpdateResource(ctx, link, request, response, options...)
}

func (d *RefDevice) UpdateResourceWithCodec(
	ctx context.Context,
	link schema.ResourceLink,
	codec coap.Codec,
	request interface{},
	response interface{},
	options ...coap.OptionFunc,
) error {
	return d.Device().UpdateResourceWithCodec(ctx, link, codec, request, response, options...)
}

func (d *RefDevice) DeleteResourceWithCodec(
	ctx context.Context,
	link schema.ResourceLink,
	codec coap.Codec,
	response interface{},
	options ...coap.OptionFunc,
) error {
	return d.Device().DeleteResourceWithCodec(ctx, link, codec, response, options...)
}

func (d *RefDevice) Own(
	ctx context.Context,
	links schema.ResourceLinks,
	otmClient []otm.Client,
	ownOptions ...core.OwnOption,
) error {
	return d.Device().Own(ctx, links, otmClient, ownOptions...)
}

func (d *RefDevice) Disown(
	ctx context.Context,
	links schema.ResourceLinks,
) error {
	return d.Device().Disown(ctx, links)
}

func (d *RefDevice) Provision(ctx context.Context, links schema.ResourceLinks) (*core.ProvisioningClient, error) {
	return d.Device().Provision(ctx, links)
}

func (d *RefDevice) GetEndpoints() []schema.Endpoint {
	return d.Device().GetEndpoints()
}

func (d *RefDevice) GetResourceLinks(ctx context.Context, endpoints []schema.Endpoint, options ...coap.OptionFunc) (schema.ResourceLinks, error) {
	return d.Device().GetResourceLinks(ctx, endpoints, options...)
}

func (d *RefDevice) FactoryReset(ctx context.Context, links schema.ResourceLinks) error {
	return d.Device().FactoryReset(ctx, links)
}

func (d *RefDevice) Reboot(ctx context.Context, links schema.ResourceLinks) error {
	return d.Device().Reboot(ctx, links)
}

func (d *RefDevice) GetOwnership(ctx context.Context, links schema.ResourceLinks) (doxm.Doxm, error) {
	return d.Device().GetOwnership(ctx, links)
}

// GetRefDevice returns device, after using call device.Release to free resources.
// Deprecated: use GetDevice instead
func (c *Client) GetRefDevice(
	ctx context.Context,
	deviceID string,
	opts ...GetDeviceOption,
) (*RefDevice, schema.ResourceLinks, error) {
	d, links, err := c.GetDevice(ctx, deviceID, opts...)
	if err != nil {
		return nil, nil, err
	}
	return NewRefDevice(d), links, nil
}

// GetRefDeviceByIP gets the device directly via IP address and multicast listen port 5683. After using it, call device.Release to free resources.
// Deprecated: use GetDeviceByIPWithLinks instead
func (c *Client) GetRefDeviceByIP(
	ctx context.Context,
	ip string,
) (*RefDevice, schema.ResourceLinks, error) {
	d, links, err := c.GetDeviceByIPWithLinks(ctx, ip)
	if err != nil {
		return nil, nil, err
	}
	return NewRefDevice(d), links, nil
}
