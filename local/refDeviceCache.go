package local

import (
	"context"
	"fmt"
	"sync"
	"time"

	kitSync "github.com/plgd-dev/kit/sync"

	cache "github.com/patrickmn/go-cache"
)

type refDeviceCache struct {
	temporaryCache     *cache.Cache
	temporaryCacheLock sync.Mutex

	permanentCache     map[string]*refCacheDevice // map[deviceID]
	permanentCacheLock sync.Mutex
}

type refCacheDevice struct {
	*kitSync.RefCounter
}

func (r *refCacheDevice) device() *RefDevice {
	return r.Data().(*RefDevice)
}

func NewRefDeviceCache(cacheExpiration time.Duration, errors func(error)) *refDeviceCache {
	cache := cache.New(cacheExpiration, time.Minute)
	cache.OnEvicted(func(key string, d interface{}) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		err := d.(*RefDevice).Release(ctx)
		if err != nil {
			errors(err)
		}
	})
	return &refDeviceCache{
		temporaryCache: cache,
		permanentCache: make(map[string]*refCacheDevice),
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
	d, ok := c.temporaryCache.Get(deviceID)
	if !ok {
		return nil, false
	}
	dev := d.(*RefDevice)
	dev.Acquire()
	return dev, true
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

func (c *refDeviceCache) TryStoreDeviceToTemporaryCache(device *RefDevice) (*RefDevice, bool, error) {
	c.temporaryCacheLock.Lock()
	defer c.temporaryCacheLock.Unlock()
	deviceID := device.DeviceID()
	d, ok := c.temporaryCache.Get(deviceID)
	if ok {
		// record is already in cache
		dev := d.(*RefDevice)
		dev.Acquire()
		return dev, false, nil
	}
	err := c.temporaryCache.Add(deviceID, device, cache.DefaultExpiration)
	if err != nil {
		return nil, false, err
	}
	device.Acquire()
	return device, true, err
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

func (c *refDeviceCache) RemoveDeviceFromPermanentCache(ctx context.Context, device *RefDevice) error {
	deviceID := device.DeviceID()
	refCacheDev, ok := c.getFromPermanentCache(deviceID)
	if !ok {
		return fmt.Errorf("device not found in remove device from permanent cache")
	}
	defer refCacheDev.Release(ctx)

	dev := refCacheDev.device()

	if dev == device {
		//remove device from cache
		return refCacheDev.Release(ctx)
	}

	return fmt.Errorf("device not found in remove device from permanent cache")
}

func (c *refDeviceCache) popTemporaryCache() map[string]cache.Item {
	c.temporaryCacheLock.Lock()
	defer c.temporaryCacheLock.Unlock()
	items := c.temporaryCache.Items()
	c.temporaryCache.Flush()
	return items
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
	for _, val := range c.popTemporaryCache() {
		d := val.Object.(*RefDevice)
		err := d.Release(ctx)
		if err != nil {
			errors = append(errors)
		}
	}
	for _, d := range c.getPermanentCacheDevices() {
		// release acquire from getPermanentCacheDevices
		err := d.Release(ctx)
		if err != nil {
			errors = append(errors)
		}
		// remove device from cache
		err = d.Release(ctx)
		if err != nil {
			errors = append(errors)
		}
	}
	return fmt.Errorf("%v", errors)
}
