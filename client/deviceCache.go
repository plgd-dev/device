package client

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/go-coap/v3/pkg/cache"
	"go.uber.org/atomic"
)

type DeviceCache struct {
	deviceExpiration time.Duration
	devicesCache     *cache.Cache[string, *core.Device]
	errors           func(error)

	closed atomic.Bool
	done   chan struct{}
}

// Creates a new cache for devices.
// - deviceExpiration: default expiration time for the device in the cache, 0 means infinite. The device expiration is refreshed by getting or updating the device.
// - pollInterval: pool interval for cleaning expired devices from the cache
// - errors: function for logging errors
func NewDeviceCache(deviceExpiration, pollInterval time.Duration, errors func(error)) *DeviceCache {
	done := make(chan struct{})
	cache := cache.NewCache[string, *core.Device]()
	if deviceExpiration > 0 {
		go func() {
			t := time.NewTicker(pollInterval)
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
	}
	return &DeviceCache{
		devicesCache:     cache,
		deviceExpiration: deviceExpiration,
		errors:           errors,
		done:             done,
	}
}

// This function loads the device from the cache and deletes it from the cache. To cleanup the device you have to call device.Close.
func (c *DeviceCache) LoadAndDeleteDevice(ctx context.Context, deviceID string) (*core.Device, bool) {
	d := c.devicesCache.Load(deviceID)
	if d == nil {
		return nil, false
	}
	dev := d.Data()
	c.devicesCache.Delete(deviceID)
	return dev, true
}

func (c *DeviceCache) GetDevice(deviceID string) (*core.Device, bool) {
	d := c.devicesCache.Load(deviceID)
	if d == nil {
		return nil, false
	}
	if deviceIsStoredWithExpiration(d) {
		d.ValidUntil.Store(getNextExpiration(c.deviceExpiration))
	}
	return d.Data(), true
}

func (c *DeviceCache) GetDeviceByFoundIP(ip string) *core.Device {
	var d *core.Device
	now := time.Now()
	c.devicesCache.Range(func(deviceID string, item *cache.Element[*core.Device]) bool {
		if item.IsExpired(now) {
			return true
		}
		dev := item.Data()
		if dev.FoundByIP() == ip {
			d = dev
			return false
		}
		return true
	})
	return d
}

func (c *DeviceCache) GetDeviceExpiration(deviceID string) (time.Time, bool) {
	d := c.devicesCache.Load(deviceID)
	if d == nil {
		return time.Time{}, false
	}
	return d.ValidUntil.Load(), true
}

// This function stores the device without timeout into the cache. The device can be removed from
// the cache only by invoking LoadAndDeleteDevice function and device.Close to cleanup connections. If a device with the same deviceID is already
// in the cache, the previous reference will be updated in the cache and it's expiration time will be set to infinite.
func (c *DeviceCache) UpdateOrStoreDevice(device *core.Device) (*core.Device, bool) {
	return c.updateOrStoreDevice(device, time.Time{})
}

// This function stores the device with the default timeout into the cache. If a device with the same
// deviceID is already in the cache the device will be updated and the expiration time will be reset
// only when the device has it set.
func (c *DeviceCache) UpdateOrStoreDeviceWithExpiration(device *core.Device) (*core.Device, bool) {
	return c.updateOrStoreDevice(device, getNextExpiration(c.deviceExpiration))
}

// return next time of expiration of device in cache. If expiration is 0, then return time.Time{}
func getNextExpiration(expiration time.Duration) time.Time {
	if expiration <= 0 {
		return time.Time{}
	}
	return time.Now().Add(expiration)
}

// Try to change the expiration time for the device in cache to default expiration.
func (c *DeviceCache) TryToChangeDeviceExpirationToDefault(deviceID string) bool {
	d := c.devicesCache.Load(deviceID)
	if d == nil {
		return false
	}
	if d.Data().FoundByIP() == "" {
		d.ValidUntil.Store(getNextExpiration(c.deviceExpiration))
		return true
	}
	return false
}

func deviceIsStoredWithExpiration(e *cache.Element[*core.Device]) bool {
	return !e.ValidUntil.Load().IsZero()
}

func (c *DeviceCache) updateOrStoreDevice(device *core.Device, expiration time.Time) (*core.Device, bool) {
	deviceID := device.DeviceID()

	d := c.devicesCache.Load(deviceID)
	if d != nil {
		dev := d.Data()
		dev.UpdateBy(device)

		// record is already in cache
		// if someone requirers from the device to be stored permanently (without timeout)
		// override the expiration
		if deviceIsStoredWithExpiration(d) {
			d.ValidUntil.Store(expiration)
		}
		return dev, true
	}
	// if the device was not in the cache store it
	loadedDev, loaded := c.devicesCache.LoadOrStore(deviceID, cache.NewElement(device, expiration, func(d1 *core.Device) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		err := d1.Close(ctx)
		if err != nil {
			c.errors(err)
		}
	}))
	dev := loadedDev.Data()
	if loaded {
		dev.UpdateBy(device)
		// record is already in cache
		// if someone requirers from the device to be stored permanently (without timeout)
		// override the expiration
		if deviceIsStoredWithExpiration(d) {
			loadedDev.ValidUntil.Store(expiration)
		}
		return dev, true
	}
	return dev, false
}

func (c *DeviceCache) GetDevicesFoundByIP() map[string]string {
	devices := make(map[string]string)
	now := time.Now()
	c.devicesCache.Range(func(deviceID string, item *cache.Element[*core.Device]) bool {
		if item.IsExpired(now) {
			return true
		}
		d := item.Data()
		if ip := d.FoundByIP(); ip != "" {
			devices[d.DeviceID()] = ip
		}
		return true
	})

	return devices
}

func (c *DeviceCache) Close(ctx context.Context) error {
	var errors []error
	if c.closed.CompareAndSwap(false, true) {
		close(c.done)
	}
	for _, val := range c.devicesCache.LoadAndDeleteAll() {
		err := val.Data().Close(ctx)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}
