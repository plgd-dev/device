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

	cancel    context.CancelFunc
	interval  time.Duration
	wait      func()
	onlineDeviceIDs *sync.Map
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
        onlineDeviceIDs: &sync.Map{},
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

func (o *devicesObserver) sendOnlineEvent(ctx context.Context, deviceID string) error {
    err := o.emit(ctx, deviceID, true)
    fmt.Println("sending device online")
    if err != nil {
        return err
    }
    return nil
}

func (o *devicesObserver) sendOfflineEvent(ctx context.Context, deviceID string) error {
    err := o.emit(ctx, deviceID, false)
    fmt.Println("sending device offline")
    if err != nil {
        return err
    }
    return nil
}

func (o *devicesObserver) observe(ctx context.Context) (*sync.Map, error) {
	newDevices := listDeviceIds{err: o.c.errors, devices: &sync.Map{}}
    resultDeviceIDs := &sync.Map{}

	err := o.discover(ctx, &newDevices)
	if err != nil {
		return nil, err
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		return nil, ctx.Err()
	}


    // for devices that were found but are not yet in onlineDeviceIDs 
    // list send an online event and store the deviceID to the result array
    // we will remove the device from the onlineDeviceIDs list because
    // we will send offline event for every device id that will be left
    // in the map at the end of this function
	newDevices.devices.Range(func(key, value interface{}) bool {
        _, loaded := o.onlineDeviceIDs.LoadAndDelete(key)

        if !loaded {
            o.sendOnlineEvent(ctx, key.(string))
        }
        resultDeviceIDs.LoadOrStore(key, true)
		return true
	})

    // check online status for all devices added by IP
    var wg sync.WaitGroup
    devicesByIP := o.c.GetAllDeviceIDsFoundByIP()
    wg.Add(len(devicesByIP))
    
    for deviceID, ip := range devicesByIP  {

        go func(deviceID string, ip string) {
            ipCtx, ipCancel := context.WithTimeout(context.Background(), 2 * time.Second)
            defer ipCancel()
            defer wg.Done()

            _, e := o.c.client.GetDeviceByIP(ipCtx, ip)
            if e != nil {
                fmt.Println(e)
            }
            online := (e == nil)

            _, loaded := o.onlineDeviceIDs.LoadAndDelete(deviceID)

            if online {
                resultDeviceIDs.LoadOrStore(deviceID, true)
            }

            if !loaded && online {
                resultDeviceIDs.LoadOrStore(deviceID, true)
                o.sendOnlineEvent(ipCtx, deviceID)
            }

            if loaded && !online {
                o.sendOfflineEvent(ipCtx, deviceID)
            }
        }(deviceID, ip)
    }
    
    wg.Wait()

    // for all devices left in the onlineDevicesIDs send an offline event
	o.onlineDeviceIDs.Range(func(key, value interface{}) bool {
        o.sendOfflineEvent(ctx, key.(string))
		return true
	})

	return resultDeviceIDs, nil
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
