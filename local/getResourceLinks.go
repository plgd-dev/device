package local

import (
	"context"
	"fmt"

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

func (d *Device) GetResourceLinks(ctx context.Context, options ...coap.OptionFunc) (schema.ResourceLinks, error) {
	addr, client, err := d.findBestClient()
	if err != nil {
		return nil, err
	}

	options = append(options, coap.WithAccept(gocoap.AppOcfCbor))
	var links schema.ResourceLinks
	err = client.GetResource(ctx, "/oic/res", &links, options...)
	if err != nil {
		return nil, err
	}
	return links.PatchEndpoint(addr), nil

}
