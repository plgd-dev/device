package link

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ocf/sdk/schema"
	cache "github.com/patrickmn/go-cache"
)

// Cache caches resource links.
type Cache struct {
	cache   *cache.Cache
	create  CacheCreateFunc
	refresh CacheRefreshFunc
}

// CacheCreateFunc is triggered on a miss by GetOrCreate to fill the missing item in the cache.
type CacheCreateFunc func(ctx context.Context, c *Cache, deviceID, href string) error

// CacheRefreshFunc is triggered on a miss by Scan to refresh the entire cache.
type CacheRefreshFunc func(ctx context.Context, c *Cache) error

// NewCache creates a cache with expiration.
func NewCache(expiration time.Duration, create CacheCreateFunc, refresh CacheRefreshFunc) *Cache {
	c := cache.New(expiration, expiration)
	return &Cache{cache: c, create: create, refresh: refresh}
}

// Put caches resource links keyed by the Device ID and Href.
func (c *Cache) Put(deviceID string, links ...schema.ResourceLink) {
	for _, link := range links {
		h := Href{DeviceID: deviceID, Href: link.Href}
		c.cache.Set(h.String(), link, cache.DefaultExpiration)
	}
}

// Delete returns the cached item or false otherwise.
func (c *Cache) Delete(deviceID, href string) {
	h := Href{DeviceID: deviceID, Href: href}
	c.cache.Delete(h.String())
}

// Get returns the cached item or false otherwise.
func (c *Cache) Get(deviceID, href string) (_ schema.ResourceLink, _ bool) {
	h := Href{DeviceID: deviceID, Href: href}
	if v, ok := c.cache.Get(h.String()); ok {
		return v.(schema.ResourceLink), true
	}
	return
}

// GetOrCreate returns the cached item or calls create otherwise.
func (c *Cache) GetOrCreate(ctx context.Context, deviceID, href string) (_ schema.ResourceLink, err error) {
	if c.create == nil {
		err = fmt.Errorf("link cache create not initialized")
		return
	}
	if r, ok := c.Get(deviceID, href); ok {
		return r, nil
	}
	err = c.create(ctx, c, deviceID, href)
	if err != nil {
		return
	}
	if r, ok := c.Get(deviceID, href); ok {
		return r, nil
	}
	err = fmt.Errorf("no resource info for device %s", deviceID)
	return
}

// CacheMatchFunc returns true to stop the Scan iteration.
type CacheMatchFunc = func(deviceID string, link schema.ResourceLink) bool

// Scan iterates through all cached items until a match is found.
// The cache is refreshed and reiterated otherwise.
func (c *Cache) Scan(ctx context.Context, match CacheMatchFunc) error {
	if c.refresh == nil {
		return fmt.Errorf("link cache refresh not initialized")
	}
	for i := 1; ; i++ {
		for k, v := range c.cache.Items() {
			di := MustParseHref(k).DeviceID
			link := v.Object.(schema.ResourceLink)
			if match(di, link) {
				return nil
			}
		}
		if i == 2 {
			break
		}
		err := c.refresh(ctx, c)
		if err != nil {
			return err
		}
	}
	return fmt.Errorf("no resource info matched")
}
