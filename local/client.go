package local

import (
	"context"
	"fmt"
	"sync"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/local/resource/link"
	"github.com/go-ocf/sdk/schema"
)

// Client an OCF local client.
type Client struct {
	factory      ResourceClientFactory
	conn         []*gocoap.MulticastClientConn
	observations *sync.Map
}

// Config for the OCF local client.
type Config struct {
	Protocol string
	Resource resource.Config
}

// NewClientFromConfig constructs a new OCF client.
func NewClientFromConfig(cfg Config, errors func(error)) (*Client, error) {
	conn := resource.DialDiscoveryAddresses(context.Background(), errors)
	linkCache, err := resource.NewLinkCache(cfg.Resource, conn)
	if err != nil {
		return nil, err
	}

	// Only TCP is supported at the moment.
	f := NewResourceClientFactory(cfg.Protocol, linkCache)

	return NewClient(f, conn), nil
}

func NewClient(f ResourceClientFactory, conn []*gocoap.MulticastClientConn) *Client {
	return &Client{factory: f, conn: conn, observations: &sync.Map{}}
}

func NewResourceClientFactory(protocol string, linkCache *link.Cache) ResourceClientFactory {
	switch protocol {
	case "tcp":
		return &tcpClientFactory{f: resource.NewTCPClientFactory(linkCache)}
	case "udp":
		return &udpClientFactory{f: resource.NewUDPClientFactory(linkCache)}
	default:
		panic(fmt.Errorf("unsupported resource client protocol %s", protocol))
	}
}

type resourceClient interface {
	Get(ctx context.Context, deviceID, href string, responseBody interface{}, options ...func(gocoap.Message)) error
	Post(ctx context.Context, deviceID, href string, requestBody interface{}, responseBody interface{}, options ...func(gocoap.Message)) error
	Observe(ctx context.Context, deviceID, href string, handler resource.ObservationHandler, options ...func(gocoap.Message)) (*gocoap.Observation, error)
}

type ResourceClientFactory interface {
	NewClient(c *gocoap.ClientConn, links schema.DeviceLinks, codec resource.Codec) (resourceClient, error)
	NewClientFromCache(codec resource.Codec) (resourceClient, error)
}

// tcpClientFactory converts the return type from *TCPClient to resourceClient.
type tcpClientFactory struct {
	f *resource.TCPClientFactory
}

func (w *tcpClientFactory) NewClient(c *gocoap.ClientConn, links schema.DeviceLinks, codec resource.Codec) (resourceClient, error) {
	return w.f.NewClient(c, links, codec)
}

func (w *tcpClientFactory) NewClientFromCache(codec resource.Codec) (resourceClient, error) {
	return w.f.NewClientFromCache(codec)
}

// udpClientFactory converts the return type from *UDPClient to resourceClient.
type udpClientFactory struct {
	f *resource.UDPClientFactory
}

func (w *udpClientFactory) NewClient(c *gocoap.ClientConn, links schema.DeviceLinks, codec resource.Codec) (resourceClient, error) {
	return w.f.NewClient(c, links, codec)
}

func (w *udpClientFactory) NewClientFromCache(codec resource.Codec) (resourceClient, error) {
	return w.f.NewClientFromCache(codec)
}
