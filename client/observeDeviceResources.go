package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/resources"
	"go.uber.org/atomic"
)

type DeviceResourcesObservationEvent_type uint8

const (
	DeviceResourcesObservationEvent_ADDED   DeviceResourcesObservationEvent_type = 0
	DeviceResourcesObservationEvent_REMOVED DeviceResourcesObservationEvent_type = 1
)

type DeviceResourcesObservationEvent struct {
	Link  schema.ResourceLink
	Event DeviceResourcesObservationEvent_type
}

type DeviceResourcesObservationHandler = interface {
	Handle(ctx context.Context, event DeviceResourcesObservationEvent) error
	OnClose()
	Error(err error)
}

type deviceResourcesObserver struct {
	c             *Client
	deviceID      string
	observationID atomic.String
	handler       DeviceResourcesObservationHandler

	links map[string]schema.ResourceLink
	mutex sync.Mutex
}

func newDeviceResourcesObserver(ctx context.Context, c *Client, deviceID string, handler DeviceResourcesObservationHandler) (*deviceResourcesObserver, error) {
	h := &deviceResourcesObserver{
		c:        c,
		deviceID: deviceID,
		handler:  handler,
	}
	obsID, err := c.ObserveResource(ctx, deviceID, resources.ResourceURI, &deviceResourcesObserver{
		c:        c,
		deviceID: deviceID,
		handler:  handler,
	})
	if err != nil {
		return nil, err
	}
	h.observationID.Store(obsID)
	return h, nil
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

	added, removed := o.processLinks(links)
	for _, l := range added {
		err := o.emit(ctx, l, true)
		if err != nil {
			_, err := o.c.StopObservingResource(ctx, o.observationID.Load())
			if err != nil {
				o.c.errors(fmt.Errorf("cannot stop observing device(%v) resources(%v): %w", o.deviceID, o.observationID.Load(), err))
			}
			return
		}
	}
	for _, l := range removed {
		err := o.emit(ctx, l, false)
		if err != nil {
			_, err := o.c.StopObservingResource(ctx, o.observationID.Load())
			if err != nil {
				o.c.errors(fmt.Errorf("cannot stop observing device(%v) resources(%v): %w", o.deviceID, o.observationID.Load(), err))
			}
			return
		}
	}
}

func (o *deviceResourcesObserver) processLinks(links schema.ResourceLinks) (added schema.ResourceLinks, removed schema.ResourceLinks) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	current := make(map[string]schema.ResourceLink)
	for _, l := range links {
		current[l.Href] = l
	}
	added = make(schema.ResourceLinks, 0, len(links))
	removed = make(schema.ResourceLinks, 0, len(links))
	for href, l := range o.links {
		_, ok := current[href]
		if !ok {
			removed = append(removed, l)
		}
	}
	for href, l := range current {
		_, ok := o.links[href]
		if !ok {
			added = append(added, l)
		}
	}
	o.links = current
	return
}

func (o *deviceResourcesObserver) emit(ctx context.Context, link schema.ResourceLink, added bool) error {
	ev := DeviceResourcesObservationEvent_REMOVED
	if added {
		ev = DeviceResourcesObservationEvent_ADDED
	}
	return o.handler.Handle(ctx, DeviceResourcesObservationEvent{
		Link:  link,
		Event: ev,
	})
}

func (c *Client) ObserveDeviceResources(ctx context.Context, deviceID string, handler DeviceResourcesObservationHandler) (string, error) {
	obs, err := newDeviceResourcesObserver(ctx, c, deviceID, handler)
	if err != nil {
		return "", err
	}
	return obs.observationID.Load(), nil
}

func (c *Client) StopObservingDeviceResources(ctx context.Context, observationID string) (bool, error) {
	return c.StopObservingResource(ctx, observationID)
}
