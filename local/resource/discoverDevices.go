package resource

import (
	"context"
	"fmt"
	"sync"

	coap "github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/kit/net"
	"github.com/go-ocf/sdk/schema"

	gocoap "github.com/go-ocf/go-coap"
)

// See the section 10.4 on the line 2482 of the Core specification:
// https://openconnectivity.org/specs/OCF_Core_Specification_v2.0.0.pdf
// https://iotivity.org/documentation/linux/programmers-guide
var (
	discoveryAddressUDP4 = []string{"224.0.1.187:5683"}
	discoveryAddressUDP6 = []string{"[ff02::158]:5683", "[ff03::158]:5683", "[ff05::158]:5683"}
)

// DialDiscoveryAddresses connects to discovery endpoints.
func DialDiscoveryAddresses(ctx context.Context, errors func(error)) []*gocoap.MulticastClientConn {
	var out []*gocoap.MulticastClientConn
	for _, address := range discoveryAddressUDP4 {
		client := gocoap.MulticastClient{Net: "udp4"}
		conn, err := client.DialWithContext(ctx, address)
		if err != nil && errors != nil {
			errors(err)
		}
		out = append(out, conn)
	}
	for _, address := range discoveryAddressUDP6 {
		client := gocoap.MulticastClient{Net: "udp6"}
		conn, err := client.DialWithContext(ctx, address)
		if err != nil && errors != nil {
			errors(err)
		}
		out = append(out, conn)
	}
	return out
}

// DiscoveryHandler receives device links and errors from the discovery multicast request.
type DiscoveryHandler interface {
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
	handler DiscoveryHandler,
) error {
	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errors := make(chan error)

	runDiscovery := runDiscovery(&wg, typeFilter, handleResponse(ctx, handler), errors)
	for _, c := range conn {
		runDiscovery(ctx, c)
	}

	select {
	case err := <-errors:
		return err
	case <-ctx.Done():
		return nil
	}
}

func runDiscovery(
	wg *sync.WaitGroup,
	typeFilter []string,
	handler func(*gocoap.Request),
	errors chan<- error,
) func(ctx context.Context, conn *gocoap.MulticastClientConn) {
	return func(ctx context.Context, conn *gocoap.MulticastClientConn) {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req, err := conn.NewGetRequest("/oic/res")
			if err != nil {
				errors <- fmt.Errorf("device discovery request creation failed: %v", err)
				return
			}

			// See "7.10.2 Use of multiple parameters within a query" in
			// https://openconnectivity.org/specs/OCF_Core_Specification_v2.0.0.pdf
			for _, t := range typeFilter {
				req.AddOption(gocoap.URIQuery, "rt="+t)
			}

			waiter, err := conn.PublishMsgWithContext(ctx, req, handler)
			if err != nil {
				errors <- fmt.Errorf("device discovery multicast request failed: %v", err)
				return
			}
			defer waiter.Cancel()

			select {
			case <-ctx.Done():
			}
		}()
	}
}

func handleResponse(ctx context.Context, handler DiscoveryHandler) func(req *gocoap.Request) {
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
