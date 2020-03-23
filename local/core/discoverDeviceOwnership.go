package core

import (
	"context"
	"fmt"

	"github.com/go-ocf/go-coap/codes"

	"github.com/go-ocf/kit/codec/ocf"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/schema"

	gocoap "github.com/go-ocf/go-coap"
)

// DiscoverDeviceOwnershipHandler receives devices ownership info.
type DiscoverDeviceOwnershipHandler interface {
	Handle(ctx context.Context, client *gocoap.ClientConn, device schema.Doxm)
	Error(err error)
}

// DiscoverOwnershipStatus type of discover ownership status.
type DiscoverOwnershipStatus int

const (
	// DiscoverAllDevices discovers owned and disowned devices.
	DiscoverAllDevices = DiscoverOwnershipStatus(0)
	// DiscoverOwnedDevices discovers owned devices,
	DiscoverOwnedDevices = DiscoverOwnershipStatus(1)
	// DiscoverDisownedDevices discovers disowned devices,
	DiscoverDisownedDevices = DiscoverOwnershipStatus(2)
)

// DiscoverDeviceOwnership discovers devices using a CoAP multicast request via UDP.
// It waits for device responses until the context is canceled.
func DiscoverDeviceOwnership(
	ctx context.Context,
	conn []*gocoap.MulticastClientConn,
	status DiscoverOwnershipStatus,
	handler DiscoverDeviceOwnershipHandler,
) error {
	var opt kitNetCoap.OptionFunc
	switch status {
	case DiscoverAllDevices:
		return Discover(ctx, conn, "/oic/sec/doxm", handleDiscoverOwnershipResponse(ctx, handler))
	case DiscoverOwnedDevices:
		opt = func(m gocoap.Message) { m.AddOption(gocoap.URIQuery, "Owned=TRUE") }
	case DiscoverDisownedDevices:
		opt = func(m gocoap.Message) { m.AddOption(gocoap.URIQuery, "Owned=FALSE") }
	default:
		return fmt.Errorf("unsupported DiscoverOwnershipStatus(%v)", status)
	}

	return Discover(ctx, conn, "/oic/sec/doxm", handleDiscoverOwnershipResponse(ctx, handler), opt)
}

func handleDiscoverOwnershipResponse(ctx context.Context, handler DiscoverDeviceOwnershipHandler) func(req *gocoap.Request) {
	return func(req *gocoap.Request) {
		if req.Msg.Code() != codes.Content {
			handler.Error(fmt.Errorf("request failed: %s", ocf.Dump(req.Msg)))
			return
		}

		var doxm schema.Doxm
		var codec ocf.VNDOCFCBORCodec
		err := codec.Decode(req.Msg, &doxm)
		if err != nil {
			handler.Error(fmt.Errorf("decoding %v failed: %w", ocf.DumpHeader(req.Msg), err))
			return
		}
		handler.Handle(ctx, req.Client, doxm)
	}
}
