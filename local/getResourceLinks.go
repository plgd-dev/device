package local

import (
	"context"
	"fmt"
	"sync"
	"time"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/net"
	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/schema"
)

func (d *Device) findBestClient() (net.Addr, *coap.Client, error) {
	var client *coap.Client
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
		switch addr.GetScheme() {
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

func operationWithRetries(parentCtx context.Context, retryFunc RetryFunc, operationTimeout time.Duration, op func(context.Context) error) error {
	for {
		ctx, cancel := context.WithTimeout(parentCtx, operationTimeout)
		opErr := op(ctx)
		cancel()
		if opErr == nil {
			return nil
		}
		when, err := retryFunc()
		if err != nil {
			return fmt.Errorf("%v: %v", err, opErr)
		}
		sleep := when.Sub(time.Now())
		if sleep > 0 {
			select {
			case <-parentCtx.Done():
				return parentCtx.Err()
			case <-time.After(sleep):
			}
		}
	}
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

func (h *deviceDiscoveryHandler) Handle(ctx context.Context, conn *gocoap.ClientConn, links schema.ResourceLinks) {
	defer conn.Close()
	h.lock.Lock()
	defer h.lock.Unlock()
	addr, err := net.Parse(schema.UDPScheme, conn.RemoteAddr())
	if err != nil {
		return
	}
	link, ok := links.GetResourceLink("/oic/d")
	if !ok {
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

func getResourceLinks(ctx context.Context, retryFunc RetryFunc, retrieveTimeout time.Duration, addr net.Addr, client *coap.Client, options ...coap.OptionFunc) (schema.ResourceLinks, error) {
	options = append(options, coap.WithAccept(gocoap.AppOcfCbor))
	var links schema.ResourceLinks

	err := operationWithRetries(ctx, retryFunc, retrieveTimeout, func(opCtx context.Context) error {
		return client.GetResource(opCtx, "/oic/res", &links, options...)
	})

	if err != nil {
		return nil, err
	}
	return links.PatchEndpoint(addr), nil
}

func (d *Device) GetResourceLinks(ctx context.Context, options ...coap.OptionFunc) (schema.ResourceLinks, error) {
	addr, client, err := d.findBestClient()
	if err == nil {
		return getResourceLinks(ctx, d.retryFunc, d.retrieveTimeout, addr, client, options...)
	}

	resLinksCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var links schema.ResourceLinks
	var ok bool

	multicastConn := DialDiscoveryAddresses(ctx, d.errFunc)
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()

	h := newDeviceDiscoveryHandler(d.DeviceID(), cancel)
	DiscoverDevices(resLinksCtx, multicastConn, h, options...)
	links, ok = h.ResourceLinks()
	if ok {
		return links, nil
	}

	return nil, fmt.Errorf("device %v not found", d.DeviceID())

}
