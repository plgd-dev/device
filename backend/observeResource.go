package backend

import (
	"context"

	"github.com/go-ocf/grpc-gateway/pb"
	codecOcf "github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	ocf "github.com/go-ocf/sdk/local"
	"github.com/gofrs/uuid"
)

func (c *Client) ObserveResourceWithCodec(
	ctx context.Context,
	deviceID string,
	href string,
	codec kitNetCoap.Codec,
	handler ocf.ObservationHandler,
) (observationID string, _ error) {
	ID, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	sub, err := c.NewResourceSubscription(ctx, pb.ResourceId{
		DeviceId:         deviceID,
		ResourceLinkHref: href,
	}, &observationHandler{
		codec: codec,
		obs:   handler,
		removeSubscription: func() {
			c.stopObservingResource(ID.String())
		},
	})
	if err != nil {
		return "", err
	}
	c.insertSubscription(ID.String(), sub)

	return ID.String(), err
}

func (c *Client) ObserveResource(
	ctx context.Context,
	deviceID string,
	href string,
	handler ocf.ObservationHandler,
) (observationID string, _ error) {
	var codec codecOcf.VNDOCFCBORCodec
	return c.ObserveResourceWithCodec(ctx, deviceID, href, codec, handler)
}

func (c *Client) stopObservingResource(observationID string) (wait func(), err error) {
	s, err := c.popSubscription(observationID)
	if err != nil {
		return nil, err
	}
	return s.Cancel()
}

func (c *Client) StopObservingResource(ctx context.Context, observationID string) error {
	wait, err := c.stopObservingResource(observationID)
	if err != nil {
		return err
	}
	wait()
	return nil
}

type observationHandler struct {
	obs                ocf.ObservationHandler
	codec              kitNetCoap.Codec
	removeSubscription func()
}

func (o *observationHandler) HandleResourceContentChanged(ctx context.Context, ev *pb.Event_ResourceChanged) error {
	o.obs.Handle(ctx, func(v interface{}) error {
		return DecodeContentWithCodec(o.codec, ev.GetContent().GetContentType(), ev.GetContent().GetData(), v)
	})
	return nil
}

func (o *observationHandler) OnClose() {
	o.removeSubscription()
	o.obs.OnClose()
}

func (o *observationHandler) Error(err error) {
	o.removeSubscription()
	o.obs.Error(err)
}
