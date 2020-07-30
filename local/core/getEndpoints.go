package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-ocf/go-coap/v2/udp/client"
	"github.com/go-ocf/kit/net"
	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/schema"
)

func (d *Device) findBestClient() (net.Addr, *coap.ClientCloseHandler, error) {
	var client *coap.ClientCloseHandler
	var addr net.Addr
	var err error

	d.lock.Lock()
	defer d.lock.Unlock()
	for key, conn := range d.conn {
		ep := schema.Endpoint{
			URI: key,
		}
		addr, err = ep.GetAddr()
		if err != nil {
			continue
		}
		switch schema.Scheme(addr.GetScheme()) {
		case schema.TCPSecureScheme:
			return addr, conn, nil
		case schema.UDPSecureScheme:
			return addr, conn, nil
		default:
			client = conn
		}
	}
	if client == nil {
		return addr, nil, fmt.Errorf("cannot find connection to device")
	}
	return addr, client, nil
}

func newDeviceDiscoveryHandler(
	deviceID string,
	cancel context.CancelFunc,
) *deviceDiscoveryHandler {
	return &deviceDiscoveryHandler{
		deviceID: deviceID,
		cancel:   cancel,
	}
}

type deviceDiscoveryHandler struct {
	deviceID string
	cancel   context.CancelFunc

	lock  sync.Mutex
	links schema.ResourceLinks
	ok    bool
}

func (h *deviceDiscoveryHandler) ResourceLinks() (schema.ResourceLinks, bool) {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.links, h.ok
}

func (h *deviceDiscoveryHandler) Handle(ctx context.Context, conn *client.ClientConn, links schema.ResourceLinks) {
	defer conn.Close()
	h.lock.Lock()
	defer h.lock.Unlock()
	addr, err := net.Parse(string(schema.UDPScheme), conn.RemoteAddr())
	if err != nil {
		return
	}
	link, err := GetResourceLink(links, "/oic/d")
	if err != nil {
		return
	}
	if h.ok || link.GetDeviceID() != h.deviceID {
		return
	}
	h.links = links.PatchEndpoint(addr)
	h.ok = true
	h.cancel()
}

func (h *deviceDiscoveryHandler) Error(err error) {
}

func (d *Device) GetEndpoints(ctx context.Context) ([]schema.Endpoint, error) {
	addr, client, err := d.findBestClient()
	if err == nil {
		links, err := getResourceLinks(ctx, addr, client, coap.WithResourceType(schema.DeviceResourceType))
		if err != nil {
			return nil, fmt.Errorf("cannot get resource links for %v: %w", d.DeviceID(), err)
		}
		dlink, err := GetResourceLink(links, "/oic/d")
		if err != nil {
			return nil, fmt.Errorf("cannot read device link for device %s: %w", d.DeviceID(), err)
		}
		return dlink.Endpoints, nil
	}
	resLinksCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var links schema.ResourceLinks
	var ok bool

	multicastConn := DialDiscoveryAddresses(ctx, d.cfg.discoveryConfiguration, d.cfg.errFunc)
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()

	h := newDeviceDiscoveryHandler(d.DeviceID(), cancel)
	DiscoverDevices(resLinksCtx, multicastConn, h, coap.WithResourceType(schema.DeviceResourceType))
	links, ok = h.ResourceLinks()
	if ok {
		dlink, err := GetResourceLink(links, "/oic/d")
		if err != nil {
			return nil, fmt.Errorf("cannot read device link for device %s: %w", d.DeviceID(), err)
		}
		return dlink.Endpoints, nil
	}

	return nil, fmt.Errorf("device %v not found", d.DeviceID())
}
