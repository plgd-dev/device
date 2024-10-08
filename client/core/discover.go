// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/device/v2/internal/math"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/net"
	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/plgd-dev/go-coap/v3/options"
	"github.com/plgd-dev/go-coap/v3/udp"
	"github.com/plgd-dev/go-coap/v3/udp/client"
	coapUdpServer "github.com/plgd-dev/go-coap/v3/udp/server"
)

// See the section 10.4 on the line 2482 of the Core specification:
// https://openconnectivity.org/specs/OCF_Core_Specification_v2.0.0.pdf
// https://iotivity.org/documentation/linux/programmers-guide
const (
	DiscoveryAddressUDP4Local      = "224.0.1.187:5683"
	DiscoveryAddressUDP6LinkLocal  = "[ff02::158]:5683"
	DiscoveryAddressUDP6RealmLocal = "[ff03::158]:5683"
	DiscoveryAddressUDP6SiteLocal  = "[ff05::158]:5683"
)

var (
	DiscoveryAddressUDP4 = []string{DiscoveryAddressUDP4Local}
	DiscoveryAddressUDP6 = []string{DiscoveryAddressUDP6LinkLocal, DiscoveryAddressUDP6RealmLocal, DiscoveryAddressUDP6SiteLocal}
)

type DiscoveryHandler = func(conn *client.Conn, req *pool.Message)

type DiscoveryClient struct {
	mcastaddr string
	msgID     uint16
	l         *net.UDPConn
	server    *coapUdpServer.Server
	wg        sync.WaitGroup
	opts      []net.MulticastOption
}

func newDiscoveryClient(network, mcastaddr string, msgID uint16, timeout time.Duration, errors func(error), opts []net.MulticastOption) (*DiscoveryClient, error) {
	l, err := net.NewListenUDP(network, "", net.WithErrors(errors))
	if err != nil {
		return nil, err
	}
	s := udp.NewServer(options.WithErrors(errors), options.WithBlockwise(true, blockwise.SZX1024, timeout), options.WithMessagePool(pool.New(0, 0)))
	c := &DiscoveryClient{
		mcastaddr: mcastaddr,
		msgID:     msgID,
		server:    s,
		l:         l,
		opts:      opts,
	}
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		err := s.Serve(l)
		if err != nil {
			errors(err)
		}
	}()
	return c, nil
}

func (d *DiscoveryClient) PublishMsgWithContext(req *pool.Message, discoveryHandler DiscoveryHandler) error {
	req.SetMessageID(int32(d.msgID))
	req.SetType(message.NonConfirmable)
	return d.server.DiscoveryRequest(req, d.mcastaddr, discoveryHandler, d.opts...)
}

func (d *DiscoveryClient) Close() error {
	d.server.Stop()
	err := d.l.Close()
	d.wg.Wait()
	return err
}

// See the section 12.2.9 https://openconnectivity.org/specs/OCF_Core_Specification.pdf
var defaultHopLimit = map[string]int{
	DiscoveryAddressUDP4Local:      1,
	DiscoveryAddressUDP6LinkLocal:  1,
	DiscoveryAddressUDP6RealmLocal: 255,
	DiscoveryAddressUDP6SiteLocal:  255,
}

func getHopLimit(addr string, desiredHopLimit int) int {
	if desiredHopLimit > 0 {
		return desiredHopLimit
	}
	if v, ok := defaultHopLimit[addr]; ok {
		return v
	}
	return 1
}

// DialDiscoveryAddresses connects to discovery endpoints.
func DialDiscoveryAddresses(ctx context.Context, cfg DiscoveryConfiguration, errFn func(error)) ([]*DiscoveryClient, error) {
	v, ok := ctx.Deadline()
	if !ok {
		return nil, errors.New("context has not set deadline")
	}
	timeout := time.Until(v)

	out := make([]*DiscoveryClient, 0, len(cfg.MulticastAddressUDP4)+len(cfg.MulticastAddressUDP6))

	// We need to separate messageIDs for upd4 and udp6, because if any docker container has isolated network
	// iotivity-lite gets error EINVAL(22) for sendmsg with UDP6 for some interfaces. If it happens, the device is
	// not discovered and msgid is cached so all other multicast messages from another interfaces are dropped for deduplication.
	msgIDudp4 := math.CastTo[uint16](message.GetMID())
	msgIDudp6 := msgIDudp4 + ^uint16(0)/2

	for _, address := range cfg.MulticastAddressUDP4 {
		multicastOptions := []net.MulticastOption{
			net.WithMulticastHoplimit(getHopLimit(address, cfg.MulticastHopLimit)),
		}
		multicastOptions = append(multicastOptions, cfg.MulticastOptions...)
		c, err := newDiscoveryClient("udp4", address, msgIDudp4, timeout, errFn, multicastOptions)
		if err != nil {
			errFn(err)
			continue
		}
		out = append(out, c)
	}
	for _, address := range cfg.MulticastAddressUDP6 {
		multicastOptions := []net.MulticastOption{
			net.WithMulticastHoplimit(getHopLimit(address, cfg.MulticastHopLimit)),
		}
		multicastOptions = append(multicastOptions, cfg.MulticastOptions...)
		c, err := newDiscoveryClient("udp6", address, msgIDudp6, timeout, errFn, multicastOptions)
		if err != nil {
			errFn(err)
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// Discover discovers devices using a CoAP multicast request via UDP.
// It waits for device responses until the context is canceled.
// Device resources can be queried in DiscoveryHandler.
// An empty typeFilter queries all resource types.
// Note: Iotivity 1.3 which responds with BadRequest if more than 1 resource type is queried.
func Discover(
	ctx context.Context,
	conn []*DiscoveryClient,
	href string,
	handler DiscoveryHandler,
	options ...coap.OptionFunc,
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
	handler DiscoveryHandler,
	errors chan<- error,
	options ...coap.OptionFunc,
) func(ctx context.Context, conn *DiscoveryClient) {
	return func(ctx context.Context, conn *DiscoveryClient) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			opts := make(message.Options, 0, 4)
			for _, o := range options {
				opts = o(opts)
			}
			req := pool.NewMessage(ctx)
			token, err := message.GetToken()
			if err != nil {
				errors <- MakeInternal(fmt.Errorf("device discovery request cannot get token: %w", err))
				return
			}
			if err = req.SetupGet(href, token, opts...); err != nil {
				errors <- MakeInternal(fmt.Errorf("device discovery request creation failed: %w", err))
				return
			}

			err = conn.PublishMsgWithContext(req, handler)
			if err != nil {
				select {
				case errors <- MakeInternal(fmt.Errorf("device discovery multicast request failed: %w", err)):
				case <-ctx.Done():
				}
				return
			}

			<-ctx.Done()
		}()
	}
}
