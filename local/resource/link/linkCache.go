package link

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/go-ocf/sdk/schema"
)

// Cache caches resource links.
type Cache struct {
	cache map[string]schema.ResourceLink
	lock  sync.Mutex

	create  CacheCreateFunc
	refresh CacheRefreshFunc
}

// CacheCreateFunc is triggered on a miss by GetOrCreate to fill the missing item in the cache.
type CacheCreateFunc func(ctx context.Context, deviceID, href string) (schema.ResourceLink, error)

// CacheRefreshFunc is triggered on a miss by Scan to refresh the entire cache.
type CacheRefreshFunc func(ctx context.Context) (*Cache, error)

// NewCache creates a cache.
func NewCache(create CacheCreateFunc, refresh CacheRefreshFunc) *Cache {
	return &Cache{
		cache:   make(map[string]schema.ResourceLink),
		create:  create,
		refresh: refresh,
	}
}

func mergeEndpoints(dest []schema.Endpoint, src []schema.Endpoint) []schema.Endpoint {
	eps := make([]schema.Endpoint, 0, len(dest)+len(src))
	eps = append(eps, dest...)
	eps = append(eps, src...)
	sort.SliceStable(eps, func(i, j int) bool { return eps[i].URI < eps[j].URI })
	sort.SliceStable(eps, func(i, j int) bool { return eps[i].Priority < eps[j].Priority })
	out := make([]schema.Endpoint, 0, len(eps))
	var last string
	for _, e := range eps {
		if last != e.URI {
			out = append(out, e)
		}
		last = e.URI
	}
	return out
}

func (c *Cache) updateLocked(deviceID string, links ...schema.ResourceLink) {
	for _, link := range links {
		h := Href{DeviceID: deviceID, Href: link.Href}
		cacheLink, ok := c.cache[h.String()]
		if ok {
			link.Endpoints = mergeEndpoints(cacheLink.Endpoints, link.Endpoints)
		}
		c.cache[h.String()] = link
	}
}

// Put caches resource links keyed by the Device ID and Href.
func (c *Cache) Update(deviceID string, links ...schema.ResourceLink) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.updateLocked(deviceID, links...)
}

// Delete returns the cached item or false otherwise.
func (c *Cache) Delete(deviceID, href string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	h := Href{DeviceID: deviceID, Href: href}
	delete(c.cache, h.String())
}

func (c *Cache) getLocked(deviceID, href string) (_ schema.ResourceLink, _ bool) {
	h := Href{DeviceID: deviceID, Href: href}
	if v, ok := c.cache[h.String()]; ok {
		return v, true
	}
	return
}

// Get returns the cached item or false otherwise.
func (c *Cache) Get(deviceID, href string) (_ schema.ResourceLink, _ bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.getLocked(deviceID, href)
}

func (c *Cache) GetOrCreate(ctx context.Context, deviceID, href string) (_ schema.ResourceLink, err error) {
	switch {
	case href == "/oic/res":
		r, err := c.getOrCreate(ctx, deviceID, "/oic/d")
		if err != nil {
			return r, err
		}
		r.Href = "/oic/res"
		return r, nil
	default:
		return c.getOrCreate(ctx, deviceID, href)
	}
}

// GetOrCreate returns the cached item or calls create otherwise.
func (c *Cache) getOrCreate(ctx context.Context, deviceID, href string) (_ schema.ResourceLink, err error) {
	if c.create == nil {
		err = fmt.Errorf("link cache create not initialized")
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	c.cache = make(map[string]schema.ResourceLink)

	if r, ok := c.getLocked(deviceID, href); ok {
		return r, nil
	}
	v, err := c.create(ctx, deviceID, href)
	if err != nil {
		return
	}
	c.updateLocked(deviceID, v)
	return v, nil
}

func (c *Cache) Items() map[string]schema.ResourceLink {
	c.lock.Lock()
	defer c.lock.Unlock()
	r := make(map[string]schema.ResourceLink)
	for k, v := range c.cache {
		r[k] = v
	}
	return r
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
		for k, link := range c.Items() {
			di := MustParseHref(k).DeviceID
			if match(di, link) {
				return nil
			}
		}
		if i == 2 {
			break
		}
		newCache, err := c.refresh(ctx)
		if err != nil {
			return err
		}
		c.lock.Lock()
		c.cache = newCache.cache
		c.lock.Unlock()
	}
	return fmt.Errorf("no resource info matched")
}
