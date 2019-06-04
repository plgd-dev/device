package local

import (
	"context"
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/google/uuid"
)

// coapContentFormat values can be found here
// https://github.com/go-ocf/go-coap/blob/a643abf9bcd9c4d033e63e7530e77d0f5f57dc54/message.go#L243
func (c *Client) ObserveResource(
	ctx context.Context,
	deviceID, href string,
	interfaceFilter string,
	coapContentFormat uint16,
	handler ObservationHandler,
) (observationID string, _ error) {
	codec := coap.NoCodec{MediaType: coapContentFormat}
	h := observationHandler{handler: handler}
	return c.observeResource(ctx, deviceID, href, interfaceFilter, codec, &h)
}

type ObservationHandler interface {
	Handle(ctx context.Context, client *gocoap.ClientConn, body []byte)
	Error(err error)
}

func (c *Client) ObserveResourceCBOR(
	ctx context.Context,
	deviceID, href string,
	interfaceFilter string,
	handler resource.ObservationHandler,
) (observationID string, _ error) {
	codec := coap.CBORCodec{}
	return c.observeResource(ctx, deviceID, href, interfaceFilter, codec, handler)
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
	interfaceFilter string,
	codec resource.Codec,
	handler resource.ObservationHandler,
) (observationID string, _ error) {
	var options []func(gocoap.Message)
	if interfaceFilter != "" {
		options = append(options, func(req gocoap.Message) {
			req.AddOption(gocoap.URIQuery, "if="+interfaceFilter)
		})
	}

	client, err := c.factory.NewClientFromCache(codec)
	if err != nil {
		return "", err
	}

	obs, err := client.Observe(ctx, deviceID, href, handler, options...)
	if err != nil {
		return "", err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("observation id generation failed: %v", err)
	}
	c.observations.Store(id.String(), obs)
	return id.String(), nil
}

type observationHandler struct {
	handler ObservationHandler
}

func (h *observationHandler) Handle(ctx context.Context, client *gocoap.ClientConn, body resource.DecodeFunc) {
	var b []byte
	if err := body(&b); err != nil {
		h.handler.Error(err)
	}
	h.handler.Handle(ctx, client, b)
}

func (h *observationHandler) Error(err error) {
	h.handler.Error(err)
}
