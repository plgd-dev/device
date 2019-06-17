package local

import (
	"context"
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	"github.com/gofrs/uuid"
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
	codec := ocf.NoCodec{MediaType: coapContentFormat}
	h := observationHandler{handler: handler}
	return c.observeResource(ctx, deviceID, href, interfaceFilter, codec, &h)
}

type ObservationHandler interface {
	Handle(ctx context.Context, client *gocoap.ClientConn, body []byte)
	Error(err error)
}

func (c *Client) ObserveResourceVNDOCFCBOR(
	ctx context.Context,
	deviceID, href string,
	interfaceFilter string,
	handler kitNetCoap.ObservationHandler,
) (observationID string, _ error) {
	codec := ocf.VNDOCFCBORCodec{}
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
	codec kitNetCoap.Codec,
	handler kitNetCoap.ObservationHandler,
) (observationID string, _ error) {
	var options []kitNetCoap.OptionFunc
	if interfaceFilter != "" {
		options = append(options, func(req gocoap.Message) {
			req.AddOption(gocoap.URIQuery, "if="+interfaceFilter)
		})
	}

	client, err := c.factory.NewClientFromCache()
	if err != nil {
		return "", err
	}

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
