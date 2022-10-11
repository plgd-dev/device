package client

import (
	"context"

	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/resources"
)

type DeviceResourcesObservationHandler = interface {
	Handle(ctx context.Context, links schema.ResourceLinks) error
	OnClose()
	Error(err error)
}

type deviceResourcesObserver struct {
	c       *Client
	handler DeviceResourcesObservationHandler
}

func newDeviceResourcesObserver(ctx context.Context, c *Client, deviceID string, handler DeviceResourcesObservationHandler) (string, error) {
	observationID, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, &deviceResourcesObserver{
		c:       c,
		handler: handler,
	})
	if err != nil {
		return "", err
	}
	return observationID, nil
}

func (o *deviceResourcesObserver) Error(err error) {
	o.handler.Error(err)
}

func (o *deviceResourcesObserver) OnClose() {
	o.handler.OnClose()
}

func (o *deviceResourcesObserver) Handle(ctx context.Context, body coap.DecodeFunc) {
	var links schema.ResourceLinks
	err := body(&links)
	if err != nil {
		o.handler.Error(err)
		return
	}
	o.handler.Handle(ctx, links)
}

func (c *Client) ObserveDeviceResources(ctx context.Context, deviceID string, handler DeviceResourcesObservationHandler) (string, error) {
	observationID, err := newDeviceResourcesObserver(ctx, c, deviceID, handler)
	if err != nil {
		return "", err
	}
	return observationID, nil
}

func (c *Client) StopObservingDeviceResources(ctx context.Context, observationID string) (bool, error) {
	return c.StopObservingResource(ctx, observationID)
}
