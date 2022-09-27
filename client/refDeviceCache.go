package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	kitSync "github.com/plgd-dev/kit/v2/sync"
	"go.uber.org/atomic"
)

type refDeviceCache struct {
	cacheExpiration    time.Duration
	temporaryCache     *Cache
	temporaryCacheLock sync.Mutex
	errors             func(error)

	permanentCache     map[string]*refCacheDevice // map[deviceID]
	permanentCacheLock sync.Mutex
	closed             atomic.Bool
	done               chan struct{}
}

type refCacheDevice struct {
	*kitSync.RefCounter
}

func (r *refCacheDevice) device() *RefDevice {
	return r.Data().(*RefDevice)
}

func NewRefDeviceCache(cacheExpiration time.Duration, errors func(error)) *refDeviceCache {
	done := make(chan struct{})
	cache := NewCache()
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case now := <-t.C:
				cache.CheckExpirations(now)
			case <-done:
				return
			}
		}
	}()
	return &refDeviceCache{
		temporaryCache:  cache,
		permanentCache:  make(map[string]*refCacheDevice),
		cacheExpiration: cacheExpiration,
		errors:          errors,
		done:            done,
	}
}

func (c *refDeviceCache) getFromPermanentCache(deviceID string) (_ *refCacheDevice, ok bool) {
	c.permanentCacheLock.Lock()
	defer c.permanentCacheLock.Unlock()
	refCacheDev, ok := c.permanentCache[deviceID]
	if ok {
		refCacheDev.Acquire()
	}
	return refCacheDev, ok
}

func (c *refDeviceCache) getDeviceFromPermanentCache(ctx context.Context, deviceID string) (*RefDevice, bool) {
	refCacheDev, ok := c.getFromPermanentCache(deviceID)
	if !ok {
		return nil, false
	}
	defer refCacheDev.Release(ctx)
	dev := refCacheDev.device()
	dev.Acquire()
	return dev, true
}

func (c *refDeviceCache) getFromTemporaryCache(deviceID string) (*RefDevice, bool) {
	c.temporaryCacheLock.Lock()
	defer c.temporaryCacheLock.Unlock()
	d := c.temporaryCache.Load(deviceID)
	if d == nil {
		return nil, false
	}
	dev := d.Data().(*RefDevice)
	dev.Acquire()
	return dev, true
}

func (c *refDeviceCache) RemoveDeviceFromTemporaryCache(ctx context.Context, deviceID string, device *RefDevice) bool {
	c.temporaryCacheLock.Lock()
	defer c.temporaryCacheLock.Unlock()
	d := c.temporaryCache.Load(deviceID)
	if d == nil {
		return false
	}
	dev := d.Data().(*RefDevice)
	if device == dev {
		// remove device from cache
		c.temporaryCache.Delete(deviceID)
		return true
	}
	return false
}

func (c *refDeviceCache) RemoveDevice(ctx context.Context, deviceID string, device *RefDevice) bool {
	ok := c.RemoveDeviceFromTemporaryCache(ctx, deviceID, device)
	return c.RemoveDeviceFromPermanentCache(ctx, deviceID, device) || ok
}

func (c *refDeviceCache) GetDevice(ctx context.Context, deviceID string) (*RefDevice, bool) {
	dev, ok := c.getDeviceFromPermanentCache(ctx, deviceID)
	if ok {
		return dev, true
	}
	dev, ok = c.getFromTemporaryCache(deviceID)
	if ok {
		return dev, true
	}
	return nil, false
}

func (c *refDeviceCache) TryStoreDeviceToTemporaryCache(device *RefDevice) (*RefDevice, bool) {
	c.temporaryCacheLock.Lock()
	defer c.temporaryCacheLock.Unlock()
	deviceID := device.DeviceID()
	for {
		d := c.temporaryCache.Load(deviceID)

		if d != nil {
			// record is already in cache
			dev := d.Data().(*RefDevice)
			if d.Data().(*RefDevice).Device().IsConnected() {
				dev.Acquire()
				return dev, false
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()
				c.temporaryCache.Delete(deviceID)
				dev.Release(ctx)
				// dev.Device().Close(ctx)
			}
		}
		_, loaded := c.temporaryCache.LoadOrStore(deviceID, NewElement(device, time.Now().Add(c.cacheExpiration), func(d1 interface{}) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			err := d1.(*RefDevice).Release(ctx)
			if err != nil {
				c.errors(err)
			}
		}))
		if loaded {
			continue
		}
		device.Acquire()
		return device, true
	}
}

func (c *refDeviceCache) StoreDeviceToPermanentCache(device *RefDevice) error {
	c.permanentCacheLock.Lock()
	defer c.permanentCacheLock.Unlock()
	deviceID := device.DeviceID()
	refCacheDev, ok := c.permanentCache[deviceID]
	if ok {
		dev := refCacheDev.device()
		if dev == device {
			refCacheDev.Acquire()
			return nil
		}
		return fmt.Errorf("device is already stored in permanent cache")
	}
	device.Acquire()
	c.permanentCache[deviceID] = &refCacheDevice{
		RefCounter: kitSync.NewRefCounter(device, func(ctx context.Context, data interface{}) error {
			dev := data.(*RefDevice)
			deviceID := device.DeviceID()
			err := dev.Release(ctx)
			c.permanentCacheLock.Lock()
			defer c.permanentCacheLock.Unlock()
			delete(c.permanentCache, deviceID)
			return err
		}),
	}
	return nil
}

func (c *refDeviceCache) RemoveDeviceFromPermanentCache(ctx context.Context, deviceID string, device *RefDevice) bool {
	refCacheDev, ok := c.getFromPermanentCache(deviceID)
	if !ok {
		return false
	}
	defer refCacheDev.Release(ctx)

	dev := refCacheDev.device()

	if dev == device {
		// remove device from cache
		refCacheDev.Release(ctx)
		return true
	}

	return false
}

func (c *refDeviceCache) popTemporaryCache() map[interface{}]interface{} {
	c.temporaryCacheLock.Lock()
	defer c.temporaryCacheLock.Unlock()
	items := c.temporaryCache.PullOutAll()
	return items
}

func (c *refDeviceCache) GetContent() map[string]string {
	c.temporaryCacheLock.Lock()
	defer c.temporaryCacheLock.Unlock()
	devices := make(map[string]string)

	c.temporaryCache.Range(func(key interface{}, value interface{}) bool {
		d := value.(*RefDevice)

		if ip := d.FoundByIP(); ip != "" {
			devices[d.DeviceID()] = ip
		}
		return true
	})

	return devices
}

func (c *refDeviceCache) getPermanentCacheDevices() []*refCacheDevice {
	c.permanentCacheLock.Lock()
	defer c.permanentCacheLock.Unlock()

	devices := make([]*refCacheDevice, 0, len(c.permanentCache))
	for _, refCacheDev := range c.permanentCache {
		refCacheDev.Acquire()
		devices = append(devices, refCacheDev)
	}
	return devices
}

func (c *refDeviceCache) Close(ctx context.Context) error {
	var errors []error
	if c.closed.CompareAndSwap(false, true) {
		close(c.done)
	}
	for _, val := range c.popTemporaryCache() {
		d := val.(*RefDevice)
		err := d.Release(ctx)
		if err != nil {
			errors = append(errors, err)
		}
	}
	for _, d := range c.getPermanentCacheDevices() {
		// release acquire from getPermanentCacheDevices
		err := d.Release(ctx)
		if err != nil {
			errors = append(errors, err)
		}
		// remove device from cache
		err = d.Release(ctx)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}
