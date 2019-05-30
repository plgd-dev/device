package resource

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/sdk/local/resource/link"
	"github.com/go-ocf/sdk/schema"
)

// NewClient constructs a new OCF client.
func NewLinkCache(cfg Config, conn []*gocoap.MulticastClientConn) (*link.Cache, error) {
	if cfg.ResourceHrefExpiration == 0 {
		return nil, fmt.Errorf("invalid resource href expiration: %v", cfg.ResourceHrefExpiration)
	}
	if cfg.DiscoveryTimeout == 0 {
		return nil, fmt.Errorf("invalid discovery timeout: %v", cfg.DiscoveryTimeout)
	}
	if cfg.DiscoveryDelay == 0 {
		return nil, fmt.Errorf("invalid discovery delay: %v", cfg.DiscoveryDelay)
	}
	if cfg.Errors == nil {
		return nil, fmt.Errorf("error handler not set")
	}
	if cfg.ResourceHrefExpiration-cfg.DiscoveryTimeout < 10*time.Second {
		return nil, fmt.Errorf("resource href expiration must be greater than discovery timeout")
	}

	refresh := refreshResourceLinks(cfg, conn)
	c := link.NewCache(refreshResourceLink(refresh), refresh)

	return c, nil
}

// Config for the link cache of the OCF local client.
type Config struct {
	ResourceHrefExpiration, DiscoveryTimeout, DiscoveryDelay time.Duration

	Errors func(error)
}

type refreshFunc = func(ctx context.Context) (*link.Cache, error)

func refreshResourceLink(f refreshFunc) link.CacheCreateFunc {
	return func(ctx context.Context, deviceID, href string) (res schema.ResourceLink, err error) {
		v, err := f(ctx)
		if err != nil {
			return res, err
		}
		if v == nil {
			return res, fmt.Errorf("cannot create resource link %v%v: not found - empty cache", deviceID, href)
		}
		res, ok := v.Get(deviceID, href)
		if !ok {
			return res, fmt.Errorf("cannot create resource link %v%v: not found", deviceID, href)
		}
		return res, nil
	}
}

func refreshResourceLinks(cfg Config, conn []*gocoap.MulticastClientConn) refreshFunc {
	var mtx sync.Mutex
	var lastCacheNum uint64
	var lastCache *link.Cache

	return func(ctx context.Context) (*link.Cache, error) {
		loadBeforeLock := atomic.LoadUint64(&lastCacheNum)

		// Delay duplicate calls
		mtx.Lock()
		defer mtx.Unlock()
		loadAfterLock := atomic.LoadUint64(&lastCacheNum)
		if loadBeforeLock != loadAfterLock {
			return lastCache, nil
		}
		atomic.AddUint64(&lastCacheNum, 1)

		// Skip subsequent calls
		/*
			if time.Since(lastRefreshTime) <= cfg.DiscoveryDelay {
				return nil, nil
			}
		*/
		timeout, cancel := context.WithTimeout(ctx, cfg.DiscoveryTimeout)
		defer cancel()
		c := link.NewCache(nil, nil)
		h := refreshHandler{linkCache: c, errors: cfg.Errors}
		err := DiscoverDevices(timeout, conn, []string{}, &h)
		if err != nil {
			return nil, err
		}

		// When canceled, do not skip subsequent calls
		/*
			select {
			case <-ctx.Done():
				return c, nil
			default:
			}
		*/

		lastCache = c
		return c, nil
	}
}

type refreshHandler struct {
	linkCache *link.Cache
	errors    func(error)
}

func (h *refreshHandler) Handle(ctx context.Context, client *gocoap.ClientConn, device schema.DeviceLinks) {
	h.linkCache.Update(device.ID, device.Links...)
}

func (h *refreshHandler) Error(err error) {
	h.errors(err)
}
