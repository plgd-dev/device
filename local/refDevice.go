package local

import (
	"context"

	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/kit/sync"
	ocf "github.com/go-ocf/sdk/local/core"
	ocfschema "github.com/go-ocf/sdk/schema"
)

type refDevice struct {
	obj *sync.RefCounter
}

func NewRefDevice(dev *ocf.Device) *refDevice {
	return &refDevice{obj: sync.NewRefCounter(dev, releaseOcfDevice)}
}

func releaseOcfDevice(ctx context.Context, data interface{}) error {
	dev := data.(*ocf.Device)
	return dev.Close(ctx)
}

func (d *refDevice) Acquire() {
	d.obj.Acquire()
}

func (d *refDevice) Release(ctx context.Context) error {
	return d.obj.Release(ctx)
}

func (d *refDevice) DeviceID() string {
	return d.device().DeviceID()
}

func (d *refDevice) device() *ocf.Device {
	return d.obj.Data().(*ocf.Device)
}

func (d *refDevice) GetDeviceDetails(ctx context.Context, links ocfschema.ResourceLinks) (out DeviceDetails, _ error) {
	return getDeviceDetails(ctx, d.device(), links)
}

func (d *refDevice) GetResourceWithCodec(
	ctx context.Context,
	link ocfschema.ResourceLink,
	codec coap.Codec,
	response interface{},
	options ...coap.OptionFunc) error {
	return d.device().GetResourceWithCodec(ctx, link, codec, response, options...)
}

func (d *refDevice) ObserveResourceWithCodec(
	ctx context.Context,
	link ocfschema.ResourceLink,
	codec coap.Codec,
	handler ocf.ObservationHandler,
	options ...coap.OptionFunc,
) (observationID string, _ error) {
	return d.device().ObserveResourceWithCodec(ctx, link, codec, handler, options...)
}

func (d *refDevice) ObserveResource(
	ctx context.Context,
	link ocfschema.ResourceLink,
	handler ocf.ObservationHandler,
	options ...coap.OptionFunc,
) (observationID string, _ error) {
	return d.device().ObserveResource(ctx, link, handler, options...)
}

func (d *refDevice) StopObservingResource(
	ctx context.Context,
	observationID string,
) error {
	return d.device().StopObservingResource(ctx, observationID)
}

func (d *refDevice) IsSecured(ctx context.Context, links ocfschema.ResourceLinks) (bool, error) {
	return d.device().IsSecured(ctx, links)
}

func (d *refDevice) UpdateResource(
	ctx context.Context,
	link ocfschema.ResourceLink,
	request interface{},
	response interface{},
	options ...coap.OptionFunc,
) error {
	return d.device().UpdateResource(ctx, link, request, response, options...)
}

func (d *refDevice) UpdateResourceWithCodec(
	ctx context.Context,
	link ocfschema.ResourceLink,
	codec coap.Codec,
	request interface{},
	response interface{},
	options ...coap.OptionFunc,
) error {
	return d.device().UpdateResourceWithCodec(ctx, link, codec, request, response, options...)
}

func (d *refDevice) Own(
	ctx context.Context,
	links ocfschema.ResourceLinks,
	otmClient ocf.OTMClient,
	ownOptions ...ocf.OwnOption,
) error {
	return d.device().Own(ctx, links, otmClient, ownOptions...)
}

func (d *refDevice) Disown(
	ctx context.Context,
	links ocfschema.ResourceLinks,
) error {
	return d.device().Disown(ctx, links)
}

func (d *refDevice) Provision(ctx context.Context, links ocfschema.ResourceLinks) (*ocf.ProvisioningClient, error) {
	return d.device().Provision(ctx, links)
}

func (d *refDevice) GetResourceLinks(ctx context.Context, options ...coap.OptionFunc) (ocfschema.ResourceLinks, error) {
	return d.device().GetResourceLinks(ctx, options...)
}

func (d *refDevice) FactoryReset(ctx context.Context, links ocfschema.ResourceLinks) error {
	return d.device().FactoryReset(ctx, links)
}

func (d *refDevice) Reboot(ctx context.Context, links ocfschema.ResourceLinks) error {
	return d.device().Reboot(ctx, links)
}

func (d *refDevice) GetOwnership(ctx context.Context) (ocfschema.Doxm, error) {
	return d.device().GetOwnership(ctx)
}
