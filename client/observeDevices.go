package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/go-coap/v2/udp/client"
	"go.uber.org/atomic"
)

type DevicesObservationEvent_type uint8

const (
	DevicesObservationEvent_ONLINE  DevicesObservationEvent_type = 0
	DevicesObservationEvent_OFFLINE DevicesObservationEvent_type = 1
)

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
	c                      *Client
	handler                *devicesObservationHandler
	discoveryConfiguration core.DiscoveryConfiguration

	cancel          context.CancelFunc
	interval        time.Duration
	wait            func()
	onlineDeviceIDs map[string]bool
}

func newDevicesObserver(c *Client, interval time.Duration, discoveryConfiguration core.DiscoveryConfiguration, handler *devicesObservationHandler) *devicesObserver {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	obs := &devicesObserver{
		c:                      c,
		handler:                handler,
		interval:               interval,
		discoveryConfiguration: discoveryConfiguration,

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
		o.onlineDeviceIDs = newDeviceIDs
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
	for deviceID := range o.onlineDeviceIDs {
		_, ok := current[deviceID]
		if !ok {
			removed = append(removed, deviceID)
		}
	}
	for deviceID := range current {
		_, ok := o.onlineDeviceIDs[deviceID]
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
func (o *listDeviceIds) Handle(ctx context.Context, client *client.ClientConn, dev schema.ResourceLinks) {
	defer client.Close()
	d, ok := dev.GetResourceLink(device.ResourceURI)
	if !ok {
		return
	}
	o.devices.Store(d.GetDeviceID(), nil)
}

// Error gets errors during discovery.
func (o *listDeviceIds) Error(err error) {
	if o.err != nil {
		o.err(err)
	}
}

func (o *devicesObserver) discover(ctx context.Context, handler core.DiscoverDevicesHandler) error {
	multicastConn, err := core.DialDiscoveryAddresses(ctx, o.discoveryConfiguration, o.c.errors)
	if err != nil {
		return fmt.Errorf("could not discover devices: %w", err)
	}
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()
	// we want to just get "oic.wk.d" resource, because links will be get via unicast to /oic/res
	return core.DiscoverDevices(ctx, multicastConn, handler, coap.WithResourceType(device.ResourceType))
}

func (o *devicesObserver) observe(ctx context.Context) (map[string]bool, error) {
	newDevices := listDeviceIds{err: o.c.errors, devices: &sync.Map{}}

	// check online status for all devices added by IP
	var wg sync.WaitGroup
	devicesByIP := o.c.GetAllDeviceIDsFoundByIP()
	// we will ping devices at once including discovery
	// therefore +1 for discovery
	wg.Add(len(devicesByIP) + 1)

	discoveryError := make(chan error, 1)
	// run discovery inside go routine
	go func() {
		defer wg.Done()
		defer close(discoveryError)

		err := o.discover(ctx, &newDevices)
		if err != nil {
			discoveryError <- err
			return
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			discoveryError <- ctx.Err()
			return
		}
	}()

	// check every device online presence inside go routine
	for deviceID, ip := range devicesByIP {
		go func(deviceID string, ip string) {
			defer wg.Done()
			if _, e := o.c.getDeviceByIPWithUpdateCache(ctx, ip, deviceID); e == nil {
				newDevices.devices.LoadOrStore(deviceID, true)
			}
		}(deviceID, ip)
	}

	wg.Wait()
	if err, received := <-discoveryError; received {
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
	o.handler.close()
	o.cancel()
}

func (o *devicesObserver) Wait() {
	o.wait()
}

type devicesObservationHandler struct {
	handlerMutex sync.Mutex
	handler      DevicesObservationHandler
	isClosed     atomic.Bool

	removeSubscription func()
}

func (h *devicesObservationHandler) close() {
	h.isClosed.Store(true)
}

func (h *devicesObservationHandler) Handle(ctx context.Context, event DevicesObservationEvent) error {
	h.handlerMutex.Lock()
	defer h.handlerMutex.Unlock()
	if h.isClosed.Load() {
		return nil
	}
	return h.handler.Handle(ctx, event)
}

func (h *devicesObservationHandler) OnClose() {
	h.handlerMutex.Lock()
	defer h.handlerMutex.Unlock()
	if h.isClosed.Load() {
		return
	}
	h.isClosed.Store(true)
	h.removeSubscription()
	h.handler.OnClose()
}

func (h *devicesObservationHandler) Error(err error) {
	h.handlerMutex.Lock()
	defer h.handlerMutex.Unlock()
	if h.isClosed.Load() {
		return
	}
	h.isClosed.Store(true)
	h.removeSubscription()
	h.handler.Error(err)
}

func (c *Client) stopObservingDevices(observationID string) (sync func(), ok bool) {
	sub, err := c.popSubscription(observationID)
	if err != nil {
		return nil, false
	}
	sub.Cancel()
	return sub.Wait, true
}

func (c *Client) ObserveDevices(ctx context.Context, handler DevicesObservationHandler, opts ...ObserveDevicesOption) (string, error) {
	cfg := observeDevicesOptions{
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnObserveDevices(cfg)
	}

	ID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	obs := newDevicesObserver(c, c.observerPollingInterval, cfg.discoveryConfiguration, &devicesObservationHandler{
		handler: handler,
		removeSubscription: func() {
			c.stopObservingDevices(ID.String())
		},
	})

	c.insertSubscription(ID.String(), obs)
	return ID.String(), nil
}

func (c *Client) StopObservingDevices(ctx context.Context, observationID string) bool {
	wait, ok := c.stopObservingDevices(observationID)
	if !ok {
		return false
	}
	wait()
	return true
}
