package client

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/go-coap/v2/pkg/cache"
	"go.uber.org/atomic"
)

type DeviceCache struct {
	defaultCacheExpiration time.Duration
	devicesCache           *cache.Cache
	errors                 func(error)

	closed atomic.Bool
	done   chan struct{}
}

func NewDeviceCache(defaultCacheExpiration, interval time.Duration, errors func(error)) *DeviceCache {
	done := make(chan struct{})
	cache := cache.NewCache()
	go func() {
		t := time.NewTicker(interval)
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
	return &DeviceCache{
		devicesCache:           cache,
		defaultCacheExpiration: defaultCacheExpiration,
		errors:                 errors,
		done:                   done,
	}
}

func (c *DeviceCache) LoadAndDeleteDevice(ctx context.Context, deviceID string) (*core.Device, bool) {
	d := c.devicesCache.Load(deviceID)
	if d == nil {
		return nil, false
	}
	dev := d.Data().(*core.Device)
	c.devicesCache.Delete(deviceID)
	return dev, true
}

func (c *DeviceCache) GetDevice(deviceID string) (*core.Device, bool) {
	d := c.devicesCache.Load(deviceID)
	if d == nil {
		return nil, false
	}
	if deviceIsStoredWithExpiration(d) {
		d.ValidUntil.Store(time.Now().Add(c.defaultCacheExpiration))
	}
	return d.Data().(*core.Device), true
}

func (c *DeviceCache) GetDeviceByFoundIP(ip string) *core.Device {
	var d *core.Device
	c.devicesCache.Range(func(key, val interface{}) bool {
		dev := val.(*core.Device)
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
// in the cache, the previous reference will bet updated in the cache and it's expiration time will be set to infinite.
func (c *DeviceCache) UpdateOrStoreDevice(device *core.Device) (*core.Device, bool) {
	return c.updateOrStoreDevice(device, time.Time{})
}

// This function stores the device with the default timeout into the cache. If a device with the same
// deviceID is already in the cache the device will be updated and the expiration time will be reset
// only when the device has it set.
func (c *DeviceCache) UpdateOrStoreDeviceWithExpiration(device *core.Device) (*core.Device, bool) {
	return c.updateOrStoreDevice(device, time.Now().Add(c.defaultCacheExpiration))
}

// Try to change the expiration time for the device in cache when if found by multicast.
func (c *DeviceCache) TryToChangeDeviceExpiration(deviceID string) bool {
	d := c.devicesCache.Load(deviceID)
	if d == nil {
		return false
	}
	if d.Data().(*core.Device).FoundByIP() == "" {
		d.ValidUntil.Store(time.Now().Add(c.defaultCacheExpiration))
		return true
	}
	return false
}

func deviceIsStoredWithExpiration(e *cache.Element) bool {
	return !e.ValidUntil.Load().IsZero()
}

func (c *DeviceCache) updateOrStoreDevice(device *core.Device, expiration time.Time) (*core.Device, bool) {
	deviceID := device.DeviceID()

	d := c.devicesCache.Load(deviceID)
	if d != nil {
		dev := d.Data().(*core.Device)
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
	loadedDev, loaded := c.devicesCache.LoadOrStore(deviceID, cache.NewElement(device, expiration, func(d1 interface{}) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		err := d1.(*core.Device).Close(ctx)
		if err != nil {
			c.errors(err)
		}
	}))
	dev := loadedDev.Data().(*core.Device)
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

func (c *DeviceCache) popDevices() map[interface{}]interface{} {
	items := c.devicesCache.PullOutAll()
	return items
}

func (c *DeviceCache) GetDevicesFoundByIP() map[string]string {
	devices := make(map[string]string)

	c.devicesCache.Range(func(key, value interface{}) bool {
		d := value.(*core.Device)

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
	for _, val := range c.popDevices() {
		d := val.(*core.Device)
		err := d.Close(ctx)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%v", errors)
	}
	return nil
}
