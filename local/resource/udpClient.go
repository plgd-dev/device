package resource

import (
	"context"
	"fmt"
	"net/url"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/net"
	"github.com/go-ocf/kit/sync"
	"github.com/go-ocf/sdk/local/resource/link"
	"github.com/go-ocf/sdk/schema"
)

// UDPClientFactory maintains the shared link cache and connection pool.
type UDPClientFactory struct {
	linkCache *link.Cache
	pool      *sync.Pool
}

// NewUDPClientFactory creates the client factory.
func NewUDPClientFactory(linkCache *link.Cache) *UDPClientFactory {
	udpPool := sync.NewPool()
	udpPool.SetFactory(createUDPConnection(udpPool))
	return &UDPClientFactory{linkCache: linkCache, pool: udpPool}
}

// NewClient populates the link cache and the connection pool,
// then creates the client that uses the shared link cache and connection pool.
func (f *UDPClientFactory) NewClient(
	c *gocoap.ClientConn,
	links schema.DeviceLinks,
) (*Client, error) {
	f.linkCache.Update(links.ID, links.Links...)
	addr, err := net.Parse(schema.UDPScheme, c.RemoteAddr())
	if err != nil {
		return nil, err
	}
	f.pool.Put(addr.String(), c)
	return f.NewClientFromCache()
}

// NewClientFromCache creates the client
// that uses the shared link cache and connection pool.
func (f *UDPClientFactory) NewClientFromCache() (*Client, error) {
	c := Client{
		linkCache: f.linkCache,
		pool:      f.pool,
		getAddr:   getUDPAddr,
	}
	return &c, nil
}

func closeConnections(pool *sync.Pool, links schema.DeviceLinks) {
	for _, link := range links.Links {
		for _, endpoint := range link.GetEndpoints() {
			url, err := url.Parse(endpoint.URI)
			if err != nil {
				continue
			}
			addr, err := net.ParseURL(url)
			if err != nil {
				continue
			}
			conn, ok := pool.Delete(addr.URL())
			if ok {
				conn.(*gocoap.ClientConn).Close()
			}
		}
	}
}

func (f *UDPClientFactory) CloseConnections(links schema.DeviceLinks) {
	closeConnections(f.pool, links)
}

func getUDPAddr(r schema.ResourceLink) (net.Addr, error) {
	return r.GetUDPAddr()
}

func createUDPConnection(p *sync.Pool) sync.PoolFunc {
	return func(ctx context.Context, urlraw string) (interface{}, error) {
		closeSession := func(error) { p.Delete(urlraw) }
		const errMsg = "cannot create udp connection to %v: %v"
		url, err := url.Parse(urlraw)
		if err != nil {
			return nil, fmt.Errorf(errMsg, urlraw, err)
		}
		addr, err := net.ParseURL(url)
		if err != nil {
			return nil, fmt.Errorf(errMsg, urlraw, err)
		}
		scheme := addr.GetScheme()
		switch scheme {
		case schema.UDPScheme:
			client := gocoap.Client{Net: "udp", NotifySessionEndFunc: closeSession}
			return client.DialWithContext(ctx, addr.String())
		}
		return nil, fmt.Errorf(errMsg, urlraw, fmt.Errorf("unsupported scheme :%v", scheme))
	}
}
