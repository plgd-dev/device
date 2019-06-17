package local

import (
	"context"
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	"github.com/gofrs/uuid"
)

func (c *Client) ObserveResourceWithCodec(
	ctx context.Context,
	deviceID, href string,
	codec kitNetCoap.Codec,
	handler ObservationHandler,
	options ...kitNetCoap.OptionFunc,
) (observationID string, _ error) {
	h := observationHandler{handler: handler}
	return c.observeResource(ctx, deviceID, href, codec, &h, options...)
}

type ObservationHandler interface {
	Handle(ctx context.Context, client *gocoap.ClientConn, body []byte)
	Error(err error)
}

func (c *Client) ObserveResource(
	ctx context.Context,
	deviceID, href string,
	handler kitNetCoap.ObservationHandler,
	options ...kitNetCoap.OptionFunc,
) (observationID string, _ error) {
	codec := ocf.VNDOCFCBORCodec{}
	return c.observeResource(ctx, deviceID, href, codec, handler, options...)
}

func (c *Client) StopObservingResource(
	ctx context.Context,
	observationID string,
) error {
	v, ok := c.observations.Load(observationID)
	if !ok {
		return fmt.Errorf("unknown observation %s", observationID)
	}
	err := v.(*gocoap.Observation).CancelWithContext(ctx)
	if err != nil {
		return fmt.Errorf("could not cancel observation %s: %v", observationID, err)
	}
	c.observations.Delete(observationID)
	return nil
}

func (c *Client) observeResource(
	ctx context.Context,
	deviceID, href string,
	codec kitNetCoap.Codec,
	handler kitNetCoap.ObservationHandler,
	options ...kitNetCoap.OptionFunc,
) (observationID string, _ error) {
	client, err := c.factory.NewClientFromCache()
	if err != nil {
		return "", err
	}

	options = append(options, kitNetCoap.WithAccept(codec.ContentFormat()))

	obs, err := client.Observe(ctx, deviceID, href, codec, handler, options...)
	if err != nil {
		return "", err
	}

	id, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("observation id generation failed: %v", err)
	}
	c.observations.Store(id.String(), obs)
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
