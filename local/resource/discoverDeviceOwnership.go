package resource

import (
	"context"
	"fmt"

	coap "github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/sdk/schema"

	gocoap "github.com/go-ocf/go-coap"
)

// DiscoverDeviceOwnershipHandler receives devices ownership info.
type DiscoverDeviceOwnershipHandler interface {
	Handle(ctx context.Context, client *gocoap.ClientConn, device schema.Doxm)
	Error(err error)
}

// DiscoverDeviceOwnership discovers devices using a CoAP multicast request via UDP.
// It waits for device responses until the context is canceled.
// Device resources can be queried in DiscoveryHandler.
// An empty typeFilter queries all resource types.
// Note: len(typeFilter) > 1 does not work with Iotivity 1.3 which responds with BadRequest.
func DiscoverDeviceOwnership(
	ctx context.Context,
	conn []*gocoap.MulticastClientConn,
	owned bool,
	handler DiscoverDeviceOwnershipHandler,
) error {
	query := "Owned=FALSE"
	if owned {
		query = "Owned=TRUE"
	}

	return Discover(ctx, conn, "/oic/sec/doxm", []string{query}, handleDiscoverOwnershipResponse(ctx, handler))
}

func handleDiscoverOwnershipResponse(ctx context.Context, handler DiscoverDeviceOwnershipHandler) func(req *gocoap.Request) {
	return func(req *gocoap.Request) {
		if req.Msg.Code() != gocoap.Content {
			handler.Error(fmt.Errorf("request failed: %s", coap.Dump(req.Msg)))
			return
		}

		var doxm schema.Doxm
		var codec coap.CBORCodec
		err := codec.Decode(req.Msg, &doxm)
		if err != nil {
			handler.Error(fmt.Errorf("decoding failed: %v: %s", err, coap.DumpHeader(req.Msg)))
			return
		}
		handler.Handle(ctx, req.Client, doxm)
	}
}
