package local

import (
	"context"
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	"github.com/gofrs/uuid"
)

func (d *Device) ObserveResourceWithCodec(
	ctx context.Context,
	href string,
	codec kitNetCoap.Codec,
	handler ObservationHandler,
	options ...kitNetCoap.OptionFunc,
) (observationID string, _ error) {
	h := observationHandler{handler: handler}
	return d.observeResource(ctx, href, codec, &h, options...)
}

type ObservationHandler interface {
	Handle(ctx context.Context, client *gocoap.ClientConn, body []byte)
	Error(err error)
}

func (d *Device) ObserveResource(
	ctx context.Context,
	deviceID, href string,
	handler kitNetCoap.ObservationHandler,
	options ...kitNetCoap.OptionFunc,
) (observationID string, _ error) {
	codec := ocf.VNDOCFCBORCodec{}
	return d.observeResource(ctx, href, codec, handler, options...)
}

func (d *Device) StopObservingResource(
	ctx context.Context,
	observationID string,
) error {
	v, ok := d.observations.Load(observationID)
	if !ok {
		return fmt.Errorf("unknown observation %s", observationID)
	}
	err := v.(*gocoap.Observation).CancelWithContext(ctx)
	if err != nil {
		return fmt.Errorf("could not cancel observation %s: %v", observationID, err)
	}
	d.observations.Delete(observationID)
	return nil
}

func (d *Device) stopObservations(ctx context.Context) error {
	obs := make([]string, 0, 12)
	d.observations.Range(func(key, value interface{}) bool {
		observationID := key.(string)
		obs = append(obs, observationID)
		return false
	})
	var errors []error
	for _, observationID := range obs {
		err := d.StopObservingResource(ctx, observationID)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}

func (d *Device) observeResource(
	ctx context.Context, href string,
	codec kitNetCoap.Codec,
	handler kitNetCoap.ObservationHandler,
	options ...kitNetCoap.OptionFunc,
) (observationID string, _ error) {

	client, err := d.connect(ctx, href)

	if err != nil {
		return "", err
	}

	options = append(options, kitNetCoap.WithAccept(codec.ContentFormat()))

	obs, err := client.Observe(ctx, href, codec, handler, options...)
	if err != nil {
		return "", err
	}

	id, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("observation id generation failed: %v", err)
	}
	d.observations.Store(id.String(), obs)
	return id.String(), nil
}

type observationHandler struct {
	handler ObservationHandler
}

func (h *observationHandler) Handle(ctx context.Context, client *gocoap.ClientConn, body kitNetCoap.DecodeFunc) {
	var b []byte
	if err := body(&b); err != nil {
		h.handler.Error(err)
	}
	h.handler.Handle(ctx, client, b)
}

func (h *observationHandler) Error(err error) {
	h.handler.Error(err)
}
