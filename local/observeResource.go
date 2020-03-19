package local

import (
	"context"
	"fmt"
	"sync"

	codecOcf "github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	ocf "github.com/go-ocf/sdk/local/core"
)

func (c *Client) ObserveResourceWithCodec(
	ctx context.Context,
	deviceID string,
	href string,
	codec kitNetCoap.Codec,
	handler ocf.ObservationHandler,
) (observationID string, _ error) {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return "", err
	}
	defer d.Release(ctx)
	obsHandler := &observationHandler{
		obs:    handler,
		client: c,
	}

	link, err := ocf.GetResourceLink(links, href)
	if err != nil {
		return "", err
	}

	observationID, err = d.ObserveResourceWithCodec(ctx, link, codec, obsHandler)
	if err != nil {
		return "", err
	}

	err = c.deviceCache.StoreDeviceToPermanentCache(d)
	if err != nil {
		return "", err
	}

	d.Acquire()
	obsHandler.Set(observationID)
	c.observeDeviceCacheLock.Lock()
	defer c.observeDeviceCacheLock.Unlock()
	c.observeDeviceCache[observationID] = d

	return observationID, err
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

func (c *Client) popObserveDevice(ctx context.Context, observationID string) (*refDevice, error) {
	c.observeDeviceCacheLock.Lock()
	defer c.observeDeviceCacheLock.Unlock()
	device, ok := c.observeDeviceCache[observationID]
	if !ok {
		return nil, fmt.Errorf("cannot find observation %v", observationID)
	}
	delete(c.observeDeviceCache, observationID)
	return device, nil
}

func (c *Client) StopObservingResource(ctx context.Context, observationID string) error {
	device, err := c.popObserveDevice(ctx, observationID)
	if err != nil {
		return err
	}
	defer device.Release(ctx)

	err = device.StopObservingResource(ctx, observationID)
	err2 := c.deviceCache.RemoveDeviceFromPermanentCache(ctx, device)
	if err != nil {
		return err2
	}
	return err
}

type observationHandler struct {
	obs    ocf.ObservationHandler
	client *Client

	lock          sync.Mutex
	observationID string
}

func (o *observationHandler) Set(observationID string) {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.observationID = observationID
}

func (o *observationHandler) Get() string {
	o.lock.Lock()
	defer o.lock.Unlock()
	return o.observationID
}

func (o *observationHandler) Handle(ctx context.Context, body kitNetCoap.DecodeFunc) {
	o.obs.Handle(ctx, body)
}

func (o *observationHandler) OnClose() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o.client.StopObservingResource(ctx, o.Get())
	o.obs.OnClose()
}

func (o *observationHandler) Error(err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o.client.StopObservingResource(ctx, o.Get())
	o.obs.Error(err)
}
