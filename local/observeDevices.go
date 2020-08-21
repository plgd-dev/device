package local

import (
	"context"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/plgd-dev/sdk/local/core"
	"github.com/plgd-dev/sdk/schema"
)

type DevicesObservationEvent_type uint8

const DevicesObservationEvent_ONLINE DevicesObservationEvent_type = 0
const DevicesObservationEvent_OFFLINE DevicesObservationEvent_type = 1

type DevicesObservationEvent struct {
	DeviceID string
	Event    DevicesObservationEvent_type
}

type DevicesObservationHandler = interface {
	Handle(ctx context.Context, event DevicesObservationEvent) error
	OnClose()
	Error(err error)
}

type devicesObserver struct {
	c                  *Client
	handler            DevicesObservationHandler
	removeSubscription func()

	cancel    context.CancelFunc
	interval  time.Duration
	wait      func()
	deviceIDs map[string]bool
}

func newDevicesObserver(c *Client, interval time.Duration, handler DevicesObservationHandler) *devicesObserver {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	obs := &devicesObserver{
		c:        c,
		handler:  handler,
		interval: interval,

		cancel: cancel,
		wait:   wg.Wait,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for obs.poll(ctx) {
		}
	}()
	return obs
}

func (o *devicesObserver) poll(ctx context.Context) bool {
	pollCtx, cancel := context.WithTimeout(ctx, o.interval)
	defer cancel()
	newDeviceIDs, err := o.observe(pollCtx)
	select {
	case <-ctx.Done():
		o.handler.OnClose()
		return false
	default:
		if err != nil {
			o.handler.Error(err)
			return false
		}
		o.deviceIDs = newDeviceIDs
		return true
	}
}

func (o *devicesObserver) processDevices(devices *sync.Map) (added map[string]bool, removed []string, current map[string]bool) {
	current = make(map[string]bool)
	devices.Range(func(key, value interface{}) bool {
		current[key.(string)] = true
		return true
	})
	added = make(map[string]bool)
	removed = make([]string, 0, len(current))
	for deviceID := range o.deviceIDs {
		_, ok := current[deviceID]
		if !ok {
			removed = append(removed, deviceID)
		}
	}
	for deviceID := range current {
		_, ok := o.deviceIDs[deviceID]
		if !ok {
			added[deviceID] = true
		}
	}
	return
}

func (o *devicesObserver) emit(ctx context.Context, deviceID string, added bool) error {
	ev := DevicesObservationEvent_OFFLINE
	if added {
		ev = DevicesObservationEvent_ONLINE
	}
	return o.handler.Handle(ctx, DevicesObservationEvent{
		DeviceID: deviceID,
		Event:    ev,
	})
}

type listDeviceIds struct {
	devices *sync.Map
	err     func(err error)
}

// Handle gets a device connection and is responsible for closing it.
func (o *listDeviceIds) Handle(ctx context.Context, device *core.Device, deviceLinks schema.ResourceLinks) {
	defer device.Close(ctx)
	o.devices.Store(device.DeviceID(), nil)
}

// Error gets errors during discovery.
func (o *listDeviceIds) Error(err error) {
	o.err(err)
}

func (o *devicesObserver) observe(ctx context.Context) (map[string]bool, error) {
	newDevices := listDeviceIds{err: o.handler.Error, devices: &sync.Map{}}
	err := o.c.GetDevicesWithHandler(ctx, &newDevices)
	if err != nil {
		return nil, err
	}
	if ctx.Err() == context.Canceled {
		return nil, err
	}

	added, removed, current := o.processDevices(newDevices.devices)
	for deviceID := range added {
		err := o.emit(ctx, deviceID, true)
		if err != nil {
			return nil, err
		}
	}
	for _, deviceID := range removed {
		err := o.emit(ctx, deviceID, false)
		if err != nil {
			return nil, err
		}
	}
	return current, nil
}

func (o *devicesObserver) Cancel() {
	o.cancel()
}

func (o *devicesObserver) Wait() {
	o.wait()
}

type devicesObservationHandler struct {
	handler            DevicesObservationHandler
	removeSubscription func()
}

func (h *devicesObservationHandler) Handle(ctx context.Context, event DevicesObservationEvent) error {
	return h.handler.Handle(ctx, event)
}

func (h *devicesObservationHandler) OnClose() {
	h.removeSubscription()
	h.handler.OnClose()
}

func (h *devicesObservationHandler) Error(err error) {
	h.removeSubscription()
	h.handler.Error(err)
}

func (c *Client) stopObservingDevices(observationID string) (sync func(), err error) {
	sub, err := c.popSubscription(observationID)
	if err != nil {
		return nil, err
	}
	sub.Cancel()
	return sub.Wait, nil
}

func (c *Client) ObserveDevices(ctx context.Context, handler DevicesObservationHandler) (string, error) {
	ID, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	obs := newDevicesObserver(c, c.observerPollingInterval, &devicesObservationHandler{
		handler: handler,
		removeSubscription: func() {
			c.stopObservingDevices(ID.String())
		},
	})

	c.insertSubscription(ID.String(), obs)
	return ID.String(), nil
}

func (c *Client) StopObservingDevices(ctx context.Context, observationID string) error {
	wait, err := c.stopObservingDevices(observationID)
	if err != nil {
		return err
	}
	wait()
	return nil
}
