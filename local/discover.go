package local

import (
	"context"
	"fmt"
	"sync"

	gocoap "github.com/go-ocf/go-coap"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
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

type DiscoveryHandler func(req *gocoap.Request)

// DiscoverDevices discovers devices using a CoAP multicast request via UDP.
// It waits for device responses until the context is canceled.
// Device resources can be queried in DiscoveryHandler.
// An empty typeFilter queries all resource types.
// Note: Iotivity 1.3 which responds with BadRequest if more than 1 resource type is queried.
func Discover(
	ctx context.Context,
	conn []*gocoap.MulticastClientConn,
	href string,
	handler DiscoveryHandler,
	options ...kitNetCoap.OptionFunc,
) error {
	var wg sync.WaitGroup
	defer wg.Wait()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errors := make(chan error)

	runDiscovery := runDiscovery(&wg, href, handler, errors, options...)
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
	href string,
	handler func(*gocoap.Request),
	errors chan<- error,
	options ...kitNetCoap.OptionFunc,
) func(ctx context.Context, conn *gocoap.MulticastClientConn) {
	return func(ctx context.Context, conn *gocoap.MulticastClientConn) {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req, err := conn.NewGetRequest(href)
			if err != nil {
				errors <- fmt.Errorf("device discovery request creation failed: %v", err)
				return
			}

			for _, opt := range options {
				opt(req)
			}

			waiter, err := conn.PublishMsgWithContext(ctx, req, handler)
			if err != nil {
				select {
				case errors <- fmt.Errorf("device discovery multicast request failed: %v", err):
				case <-ctx.Done():
				}
				return
			}
			defer waiter.Cancel()

			select {
			case <-ctx.Done():
			}
		}()
	}
}
