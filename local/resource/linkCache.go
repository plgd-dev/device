package resource

import (
	"context"
	"fmt"
	"sync"
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
	c := link.NewCache(cfg.ResourceHrefExpiration, refreshResourceLink(refresh), refresh)
	return c, nil
}

// Config for the link cache of the OCF local client.
type Config struct {
	ResourceHrefExpiration, DiscoveryTimeout, DiscoveryDelay time.Duration

	Errors func(error)
}

type refreshFunc = func(ctx context.Context, c *link.Cache) error

func refreshResourceLink(f refreshFunc) link.CacheCreateFunc {
	return func(ctx context.Context, c *link.Cache, deviceID, href string) error {
		return f(ctx, c)
	}
}

func refreshResourceLinks(cfg Config, conn []*gocoap.MulticastClientConn) refreshFunc {
	var mtx sync.Mutex
	var lastRefreshTime time.Time

	return func(ctx context.Context, c *link.Cache) error {
		// Delay duplicate calls
		mtx.Lock()
		defer mtx.Unlock()
		// Skip subsequent calls
		if time.Since(lastRefreshTime) <= cfg.DiscoveryDelay {
			return nil
		}

		timeout, cancel := context.WithTimeout(ctx, cfg.DiscoveryTimeout)
		defer cancel()
		h := refreshHandler{linkCache: c, errors: cfg.Errors}
		err := DiscoverDevices(timeout, conn, []string{}, &h)
		if err != nil {
			return err
		}

		// When canceled, do not skip subsequent calls
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		lastRefreshTime = time.Now()
		return nil
	}
}

type refreshHandler struct {
	linkCache *link.Cache
	errors    func(error)
}

func (h *refreshHandler) Handle(ctx context.Context, client *gocoap.ClientConn, device schema.DeviceLinks) {
	h.linkCache.Put(device.ID, device.Links...)
}

func (h *refreshHandler) Error(err error) {
	h.errors(err)
}
