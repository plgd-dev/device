package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/go-coap/v2/pkg/cache"
	"go.uber.org/atomic"
)

type refDeviceCache struct {
	defaultCacheExpiration time.Duration
	devicesCache           *cache.Cache
	devicesCacheLock       sync.Mutex
	errors                 func(error)

	closed atomic.Bool
	done   chan struct{}
}

func NewRefDeviceCache(defaultCacheExpiration time.Duration, errors func(error)) *refDeviceCache {
	done := make(chan struct{})
	cache := cache.NewCache()
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
		devicesCache:           cache,
		defaultCacheExpiration: defaultCacheExpiration,
		errors:                 errors,
		done:                   done,
	}
}

func (c *refDeviceCache) RemoveDevice(deviceID string, device *RefDevice) bool {
	c.devicesCacheLock.Lock()
	defer c.devicesCacheLock.Unlock()
	d := c.devicesCache.Load(deviceID)
	if d == nil {
		return false
	}
	dev := d.Data().(*RefDevice)
	if device == dev {
		// remove device from cache
		c.devicesCache.Delete(deviceID)
		return true
	}
	return false
}

func (c *refDeviceCache) GetDevice(deviceID string) (*RefDevice, bool) {
	c.devicesCacheLock.Lock()
	defer c.devicesCacheLock.Unlock()
	d := c.devicesCache.Load(deviceID)
	if d == nil {
		return nil, false
	}
	dev := d.Data().(*RefDevice)
	dev.Acquire()
	return dev, true
}

func (c *refDeviceCache) GetDeviceExpiration(deviceID string) (time.Time, bool) {
	c.devicesCacheLock.Lock()
	defer c.devicesCacheLock.Unlock()
	d := c.devicesCache.Load(deviceID)
	if d == nil {
		return time.Time{}, false
	}
	return d.ValidUntil.Load(), true
}

// This function stores the device without timeout into the cache. The device can be removed from
// the cache only by invoking removeDevice function. If a device with the same deviceID is already
// in the cache, the previous reference will stay in the cache but it's expiration time will be removed.
func (c *refDeviceCache) TryStoreDeviceWithoutTimeout(device *RefDevice) (*RefDevice, bool) {
	return c.tryStoreDevice(device, time.Time{})
}

// This function stores the device with the defualt timeout into the cache. If a device with the same
// deviceID is already in the cache no changes will be invoked.
func (c *refDeviceCache) TryStoreDevice(device *RefDevice) (*RefDevice, bool) {
	return c.tryStoreDevice(device, time.Now().Add(c.defaultCacheExpiration))
}

func (c *refDeviceCache) tryStoreDevice(device *RefDevice, expiration time.Time) (*RefDevice, bool) {
	c.devicesCacheLock.Lock()
	defer c.devicesCacheLock.Unlock()
	deviceID := device.DeviceID()

	d := c.devicesCache.Load(deviceID)
	if d != nil {
		// record is already in cache
		// if someone requieres from the device to be stored permanently (without timeout)
		// override the expiration
		if !d.ValidUntil.Load().IsZero() && expiration.IsZero() {
			d.ValidUntil.Store(expiration)
		}

		dev := d.Data().(*RefDevice)
		dev.Acquire()
		return dev, false
	}

	// if the device was not in the cache store it
	loadedDev, _ := c.devicesCache.LoadOrStore(deviceID, cache.NewElement(device, expiration, func(d1 interface{}) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		err := d1.(*RefDevice).Release(ctx)
		if err != nil {
			c.errors(err)
		}
	}))
	refDev := loadedDev.Data().(*RefDevice)
	refDev.Acquire()
	return refDev, true
}

func (c *refDeviceCache) popDevices() map[interface{}]interface{} {
	c.devicesCacheLock.Lock()
	defer c.devicesCacheLock.Unlock()
	items := c.devicesCache.PullOutAll()
	return items
}

func (c *refDeviceCache) GetDevicesFoundByIP() map[string]string {
	c.devicesCacheLock.Lock()
	defer c.devicesCacheLock.Unlock()
	devices := make(map[string]string)

	c.devicesCache.Range(func(key, value interface{}) bool {
		d := value.(*RefDevice)

		if ip := d.FoundByIP(); ip != "" {
			devices[d.DeviceID()] = ip
		}
		return true
	})

	return devices
}

func (c *refDeviceCache) Close(ctx context.Context) error {
	var errors []error
	if c.closed.CompareAndSwap(false, true) {
		close(c.done)
	}
	for _, val := range c.popDevices() {
		d := val.(*RefDevice)
		err := d.Release(ctx)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}
