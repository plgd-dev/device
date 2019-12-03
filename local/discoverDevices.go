package local

import (
	"github.com/go-ocf/go-coap/codes"
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
	Handle(ctx context.Context, client *gocoap.ClientConn, device schema.ResourceLinks)
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
	options = append(options, coap.WithAccept(gocoap.AppOcfCbor))
	return Discover(ctx, conn, "/oic/res", handleResponse(ctx, handler), options...)
}

func handleResponse(ctx context.Context, handler DiscoverDevicesHandler) func(req *gocoap.Request) {
	return func(req *gocoap.Request) {
		if req.Msg.Code() != codes.Content {
			handler.Error(fmt.Errorf("request failed: %s", ocf.Dump(req.Msg)))
			return
		}

		var links schema.ResourceLinks
		var codec DiscoverDeviceCodec

		err := codec.Decode(req.Msg, &links)
		if err != nil {
			handler.Error(fmt.Errorf("decoding failed: %v: %s", err, ocf.DumpHeader(req.Msg)))
			return
		}
		addr, err := net.Parse(string(schema.UDPScheme), req.Client.RemoteAddr())
		if err != nil {
			handler.Error(fmt.Errorf("invalid address %v: %v", req.Client.RemoteAddr(), err))
			return
		}
		links = links.PatchEndpoint(addr)
		if len(links) > 0 {
			handler.Handle(ctx, req.Client, links)
		}
	}
}
