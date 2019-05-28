package resource

import (
	"context"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/net"
	"github.com/go-ocf/kit/sync"
	"github.com/go-ocf/sdk/local/resource/link"
	"github.com/go-ocf/sdk/schema"
)

// TCPClientFactory maintains the shared link cache and connection pool.
type TCPClientFactory struct {
	linkCache *link.Cache
	pool      *sync.Pool
}

// NewTCPClientFactory creates the client factory.
func NewTCPClientFactory(linkCache *link.Cache) *TCPClientFactory {
	tcpPool := sync.NewPool()
	tcpPool.SetFactory(createTCPConnection(tcpPool))
	return &TCPClientFactory{linkCache: linkCache, pool: tcpPool}
}

// NewClient populates the link cache and creates the client
// that uses the shared link cache and connection pool.
func (f *TCPClientFactory) NewClient(
	c *gocoap.ClientConn,
	links schema.DeviceLinks,
	codec Codec,
) (*Client, error) {
	f.linkCache.Put(links.ID, links.Links...)
	return f.NewClientFromCache(codec)
}

// NewClientFromCache creates the client
// that uses the shared link cache and connection pool.
func (f *TCPClientFactory) NewClientFromCache(codec Codec) (*Client, error) {
	c := Client{
		linkCache: f.linkCache,
		pool:      f.pool,
		codec:     codec,
		getAddr:   getTCPAddr,
	}
	return &c, nil
}

func getTCPAddr(r *schema.ResourceLink) (net.Addr, error) {
	return r.GetTCPAddr()
}

func createTCPConnection(p *sync.Pool) sync.PoolFunc {
	return func(ctx context.Context, addr string) (interface{}, error) {
		closeSession := func(error) { p.Delete(addr) }
		client := gocoap.Client{Net: "tcp", NotifySessionEndFunc: closeSession,
			// Iotivity 1.3 breaks with signal messages,
			// but Iotivity 2.0 requires them.
			DisableTCPSignalMessages: false,
		}
		return client.DialWithContext(ctx, addr)
	}
}
