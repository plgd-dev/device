package resource

import (
	"context"
	"fmt"

	coap "github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/kit/net"
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
// Note: len(typeFilter) > 1 does not work with Iotivity 1.3 which responds with BadRequest.
func DiscoverDevices(
	ctx context.Context,
	conn []*gocoap.MulticastClientConn,
	typeFilter []string,
	handler DiscoverDevicesHandler,
) error {
	queries := make([]string, 0, len(typeFilter))
	for _, t := range typeFilter {
		queries = append(queries, "rt="+t)
	}

	return Discover(ctx, conn, "/oic/res", queries, handleResponse(ctx, handler))
}

func handleResponse(ctx context.Context, handler DiscoverDevicesHandler) func(req *gocoap.Request) {
	return func(req *gocoap.Request) {
		if req.Msg.Code() != gocoap.Content {
			handler.Error(fmt.Errorf("request failed: %s", coap.Dump(req.Msg)))
			return
		}

		var devices []schema.DeviceLinks
		var codec DiscoveryResourceCodec

		err := codec.Decode(req.Msg, &devices)
		if err != nil {
			handler.Error(fmt.Errorf("decoding failed: %v: %s", err, coap.DumpHeader(req.Msg)))
			return
		}

		for _, device := range devices {
			if device.ID == "00000000-cafe-baba-0000-000000000000" {
				fmt.Println(coap.Dump(req.Msg))
			}

			addr, err := net.Parse(req.Client.RemoteAddr())
			if err != nil {
				handler.Error(fmt.Errorf("invalid address of device %s: %v", device.ID, err))
				continue
			}
			device = device.PatchEndpoint(addr)

			handler.Handle(ctx, req.Client, device)
		}
	}
}
