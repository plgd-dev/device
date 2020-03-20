package local

import (
	"context"

	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/kit/sync"
	ocf "github.com/go-ocf/sdk/local/core"
	ocfschema "github.com/go-ocf/sdk/schema"
)

type RefDevice struct {
	obj *sync.RefCounter
}

func NewRefDevice(dev *ocf.Device) *RefDevice {
	return &RefDevice{obj: sync.NewRefCounter(dev, releaseOcfDevice)}
}

func releaseOcfDevice(ctx context.Context, data interface{}) error {
	dev := data.(*ocf.Device)
	return dev.Close(ctx)
}

func (d *RefDevice) Acquire() {
	d.obj.Acquire()
}

func (d *RefDevice) Release(ctx context.Context) error {
	return d.obj.Release(ctx)
}

func (d *RefDevice) DeviceID() string {
	return d.Device().DeviceID()
}

func (d *RefDevice) Device() *ocf.Device {
	return d.obj.Data().(*ocf.Device)
}

func (d *RefDevice) GetDeviceDetails(ctx context.Context, links ocfschema.ResourceLinks) (out DeviceDetails, _ error) {
	return getDeviceDetails(ctx, d.Device(), links)
}

func (d *RefDevice) GetResourceWithCodec(
	ctx context.Context,
	link ocfschema.ResourceLink,
	codec coap.Codec,
	response interface{},
	options ...coap.OptionFunc) error {
	return d.Device().GetResourceWithCodec(ctx, link, codec, response, options...)
}

func (d *RefDevice) ObserveResourceWithCodec(
	ctx context.Context,
	link ocfschema.ResourceLink,
	codec coap.Codec,
	handler ocf.ObservationHandler,
	options ...coap.OptionFunc,
) (observationID string, _ error) {
	return d.Device().ObserveResourceWithCodec(ctx, link, codec, handler, options...)
}

func (d *RefDevice) ObserveResource(
	ctx context.Context,
	link ocfschema.ResourceLink,
	handler ocf.ObservationHandler,
	options ...coap.OptionFunc,
) (observationID string, _ error) {
	return d.Device().ObserveResource(ctx, link, handler, options...)
}

func (d *RefDevice) StopObservingResource(
	ctx context.Context,
	observationID string,
) error {
	return d.Device().StopObservingResource(ctx, observationID)
}

func (d *RefDevice) IsSecured(ctx context.Context, links ocfschema.ResourceLinks) (bool, error) {
	return d.Device().IsSecured(ctx, links)
}

func (d *RefDevice) UpdateResource(
	ctx context.Context,
	link ocfschema.ResourceLink,
	request interface{},
	response interface{},
	options ...coap.OptionFunc,
) error {
	return d.Device().UpdateResource(ctx, link, request, response, options...)
}

func (d *RefDevice) UpdateResourceWithCodec(
	ctx context.Context,
	link ocfschema.ResourceLink,
	codec coap.Codec,
	request interface{},
	response interface{},
	options ...coap.OptionFunc,
) error {
	return d.Device().UpdateResourceWithCodec(ctx, link, codec, request, response, options...)
}

func (d *RefDevice) Own(
	ctx context.Context,
	links ocfschema.ResourceLinks,
	otmClient ocf.OTMClient,
	ownOptions ...ocf.OwnOption,
) error {
	return d.Device().Own(ctx, links, otmClient, ownOptions...)
}

func (d *RefDevice) Disown(
	ctx context.Context,
	links ocfschema.ResourceLinks,
) error {
	return d.Device().Disown(ctx, links)
}

func (d *RefDevice) Provision(ctx context.Context, links ocfschema.ResourceLinks) (*ocf.ProvisioningClient, error) {
	return d.Device().Provision(ctx, links)
}

func (d *RefDevice) GetResourceLinks(ctx context.Context, options ...coap.OptionFunc) (ocfschema.ResourceLinks, error) {
	return d.Device().GetResourceLinks(ctx, options...)
}

func (d *RefDevice) FactoryReset(ctx context.Context, links ocfschema.ResourceLinks) error {
	return d.Device().FactoryReset(ctx, links)
}

func (d *RefDevice) Reboot(ctx context.Context, links ocfschema.ResourceLinks) error {
	return d.Device().Reboot(ctx, links)
}

func (d *RefDevice) GetOwnership(ctx context.Context) (ocfschema.Doxm, error) {
	return d.Device().GetOwnership(ctx)
}
