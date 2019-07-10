package local

import (
	"context"
	"fmt"

	"github.com/go-ocf/kit/codec/ocf"
	"github.com/go-ocf/kit/net"
	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/schema"

	gocoap "github.com/go-ocf/go-coap"
)

// DiscoverDevicesHandler receives device links and errors from the discovery multicast request.
type DiscoverDevicesHandler interface {
	Handle(ctx context.Context, client *gocoap.ClientConn, device schema.DeviceLinks)
	Error(err error)
}

// DiscoverDevices discovers devices using a CoAP multicast request via UDP.
// It waits for device responses until the context is canceled.
// Device resources can be queried in DiscoveryHandler.
// An empty typeFilter queries all resource types.
// Note: Iotivity 1.3 which responds with BadRequest if more than 1 resource type is queried.
func DiscoverDevices(
	ctx context.Context,
	conn []*gocoap.MulticastClientConn,
	handler DiscoverDevicesHandler,
	options ...coap.OptionFunc,
) error {
	return Discover(ctx, conn, "/oic/res", handleResponse(ctx, handler), options...)
}

func FilterResourceLinksWithEndpoints(in []schema.ResourceLink) []schema.ResourceLink {
	links := make([]schema.ResourceLink, 0, len(in))
	for _, link := range in {
		if len(link.Endpoints) > 0 {
			links = append(links, link)
		}
	}
	return links
}

func handleResponse(ctx context.Context, handler DiscoverDevicesHandler) func(req *gocoap.Request) {
	return func(req *gocoap.Request) {
		if req.Msg.Code() != gocoap.Content {
			handler.Error(fmt.Errorf("request failed: %s", ocf.Dump(req.Msg)))
			return
		}

		var devices []schema.DeviceLinks
		var codec DiscoveryResourceCodec

		err := codec.Decode(req.Msg, &devices)
		if err != nil {
			handler.Error(fmt.Errorf("decoding failed: %v: %s", err, ocf.DumpHeader(req.Msg)))
			return
		}

		for _, device := range devices {
			addr, err := net.Parse("coap://", req.Client.RemoteAddr())
			if err != nil {
				handler.Error(fmt.Errorf("invalid address of device %s: %v", device.ID, err))
				continue
			}
			device = device.PatchEndpoint(addr)

			//filter device links with endpoints
			links := FilterResourceLinksWithEndpoints(device.Links)
			if len(links) > 0 {
				handler.Handle(ctx, req.Client, device)
			}
		}
	}
}
