package client

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/schema"
	"go.uber.org/atomic"
)

type DeviceResourcesObservationEvent_type uint8

const DeviceResourcesObservationEvent_ADDED DeviceResourcesObservationEvent_type = 0
const DeviceResourcesObservationEvent_REMOVED DeviceResourcesObservationEvent_type = 1

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
	c        *Client
	deviceID string
	handler  *deviceResourcesObservationHandler

	cancel   context.CancelFunc
	interval time.Duration
	wait     func()
	links    map[string]schema.ResourceLink
}

func newDeviceResourcesObserver(c *Client, deviceID string, interval time.Duration, handler *deviceResourcesObservationHandler) *deviceResourcesObserver {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	obs := &deviceResourcesObserver{
		c:        c,
		deviceID: deviceID,
		handler:  handler,
		interval: interval,

		wait:   wg.Wait,
		cancel: cancel,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for obs.poll(ctx) {
		}
	}()
	return obs
}

func (o *deviceResourcesObserver) poll(ctx context.Context) bool {
	pollCtx, cancel := context.WithTimeout(ctx, o.interval)
	defer cancel()
	newLinks, err := o.observe(pollCtx)
	select {
	case <-ctx.Done():
		o.handler.OnClose()
		return false
	case <-pollCtx.Done():
		if err != nil {
			o.handler.Error(err)
			return false
		}
		o.links = newLinks
		return true
	}
}

func (o *deviceResourcesObserver) processLinks(links schema.ResourceLinks) (added schema.ResourceLinks, removed schema.ResourceLinks, current map[string]schema.ResourceLink) {
	current = make(map[string]schema.ResourceLink)
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

func (o *deviceResourcesObserver) observe(ctx context.Context) (map[string]schema.ResourceLink, error) {
	refDev, links, err := o.c.GetRefDevice(ctx, o.deviceID)
	if err != nil {
		return nil, err
	}

	err = refDev.Release(ctx)
	if err != nil {
		return nil, err
	}

	added, removed, current := o.processLinks(links)
	for _, l := range added {
		err := o.emit(ctx, l, true)
		if err != nil {
			return nil, err
		}
	}
	for _, l := range removed {
		err := o.emit(ctx, l, false)
		if err != nil {
			return nil, err
		}
	}
	return current, nil
}

func (o *deviceResourcesObserver) Cancel() {
	o.handler.close()
	o.cancel()
}

func (o *deviceResourcesObserver) Wait() {
	o.wait()
}

type deviceResourcesObservationHandler struct {
	handlerMutex sync.Mutex
	handler      DeviceResourcesObservationHandler
	isClosed     atomic.Bool

	removeSubscription func()
}

func (h *deviceResourcesObservationHandler) close() {
	h.isClosed.Store(true)
}

func (h *deviceResourcesObservationHandler) Handle(ctx context.Context, event DeviceResourcesObservationEvent) error {
	h.handlerMutex.Lock()
	defer h.handlerMutex.Unlock()
	if h.isClosed.Load() {
		return nil
	}
	return h.handler.Handle(ctx, event)
}

func (h *deviceResourcesObservationHandler) OnClose() {
	h.handlerMutex.Lock()
	defer h.handlerMutex.Unlock()
	if h.isClosed.Load() {
		return
	}
	h.isClosed.Store(true)
	h.removeSubscription()
	h.handler.OnClose()
}

func (h *deviceResourcesObservationHandler) Error(err error) {
	h.handlerMutex.Lock()
	defer h.handlerMutex.Unlock()
	if h.isClosed.Load() {
		return
	}
	h.isClosed.Store(true)
	h.removeSubscription()
	h.handler.Error(err)
}

func (c *Client) stopObservingDeviceResources(observationID string) (sync func(), err error) {
	sub, err := c.popSubscription(observationID)
	if err != nil {
		return nil, err
	}
	sub.Cancel()
	return sub.Wait, nil
}

func (c *Client) ObserveDeviceResources(ctx context.Context, deviceID string, handler DeviceResourcesObservationHandler) (string, error) {
	ID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	obs := newDeviceResourcesObserver(c, deviceID, c.observerPollingInterval, &deviceResourcesObservationHandler{
		handler: handler,
		removeSubscription: func() {
			c.stopObservingDevices(ID.String())
		},
	})
	c.insertSubscription(ID.String(), obs)
	return ID.String(), nil
}

func (c *Client) StopObservingDeviceResources(ctx context.Context, observationID string) error {
	wait, err := c.stopObservingDeviceResources(observationID)
	if err != nil {
		return err
	}
	wait()
	return nil
}
