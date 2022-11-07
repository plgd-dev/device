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

package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/client/core/otm"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/go-coap/v3/net/blockwise"
	"github.com/plgd-dev/go-coap/v3/options"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
	"github.com/plgd-dev/go-coap/v3/tcp"
	tcpClient "github.com/plgd-dev/go-coap/v3/tcp/client"
	"github.com/plgd-dev/go-coap/v3/udp"
	udpClient "github.com/plgd-dev/go-coap/v3/udp/client"
)

type ApplicationCallback = interface {
	GetRootCertificateAuthorities() ([]*x509.Certificate, error)
	GetManufacturerCertificateAuthorities() ([]*x509.Certificate, error)
	GetManufacturerCertificate() (tls.Certificate, error)
}

type subscription = interface {
	Cancel()
	Wait()
}

type Config struct {
	DeviceCacheExpirationSeconds   int64
	ObserverPollingIntervalSeconds uint64 // 0 means 3 seconds

	KeepAliveConnectionTimeoutSeconds uint64 // 0 means keepalive is disabled
	MaxMessageSize                    uint32
	DisablePeerTCPSignalMessageCSMs   bool
	DefaultTransferDurationSeconds    uint64 // 0 means 15 seconds

	// specify one of:
	DeviceOwnershipSDK     *DeviceOwnershipSDKConfig     `yaml:",omitempty"`
	DeviceOwnershipBackend *DeviceOwnershipBackendConfig `yaml:",omitempty"`
}

// NewClientFromConfig constructs a new local client from the proto configuration.
func NewClientFromConfig(cfg *Config, app ApplicationCallback, logger core.Logger) (*Client, error) {
	var cacheExpiration time.Duration
	if cfg.DeviceCacheExpirationSeconds > 0 {
		cacheExpiration = time.Second * time.Duration(cfg.DeviceCacheExpirationSeconds)
	}

	observerPollingInterval := time.Second * 3
	if cfg.ObserverPollingIntervalSeconds > 0 {
		observerPollingInterval = time.Second * time.Duration(cfg.ObserverPollingIntervalSeconds)
	}

	tcpDialOpts := make([]tcp.Option, 0, 5)
	udpDialOpts := make([]udp.Option, 0, 5)

	if logger == nil {
		logger = core.NewNilLogger()
	}

	errFn := func(err error) {
		logger.Debug(err.Error())
	}

	tcpDialOpts = append(tcpDialOpts, options.WithErrors(errFn))
	udpDialOpts = append(udpDialOpts, options.WithErrors(errFn))

	keepAliveConnectionTimeout := time.Second * 60
	if cfg.KeepAliveConnectionTimeoutSeconds > 0 {
		keepAliveConnectionTimeout = time.Second * time.Duration(cfg.KeepAliveConnectionTimeoutSeconds)
	}
	tcpDialOpts = append(tcpDialOpts, options.WithKeepAlive(3, keepAliveConnectionTimeout/3, func(cc *tcpClient.Conn) {
		errFn(fmt.Errorf("keepalive failed for tcp: %v", cc.RemoteAddr()))
		if err := cc.Close(); err != nil {
			errFn(fmt.Errorf("failed to close tcp connection: %v", cc.RemoteAddr()))
		}
	}))
	udpDialOpts = append(udpDialOpts, options.WithKeepAlive(3, keepAliveConnectionTimeout/3, func(cc *udpClient.Conn) {
		errFn(fmt.Errorf("keepalive failed for udp: %v", cc.RemoteAddr()))
		if err := cc.Close(); err != nil {
			errFn(fmt.Errorf("failed to close udp connection: %v", cc.RemoteAddr()))
		}
	}))

	maxMessageSize := uint32(512 * 1024)
	if cfg.MaxMessageSize > 0 {
		maxMessageSize = cfg.MaxMessageSize
	}
	tcpDialOpts = append(tcpDialOpts, options.WithMaxMessageSize(maxMessageSize))
	udpDialOpts = append(udpDialOpts, options.WithMaxMessageSize(maxMessageSize))

	if cfg.DisablePeerTCPSignalMessageCSMs {
		tcpDialOpts = append(tcpDialOpts, options.WithDisablePeerTCPSignalMessageCSMs())
	}

	defaultTransferDuration := time.Second * 15
	if cfg.DefaultTransferDurationSeconds > 0 {
		defaultTransferDuration = time.Second * time.Duration(cfg.DefaultTransferDurationSeconds)
	}
	tcpDialOpts = append(tcpDialOpts, options.WithBlockwise(true, blockwise.SZX1024, defaultTransferDuration))
	udpDialOpts = append(udpDialOpts, options.WithBlockwise(true, blockwise.SZX1024, defaultTransferDuration))

	dialTLS := func(ctx context.Context, addr string, tlsCfg *tls.Config, opts ...tcp.Option) (*coap.ClientCloseHandler, error) {
		opts = append(opts, tcpDialOpts...)
		return coap.DialTCPSecure(ctx, addr, tlsCfg, opts...)
	}
	dialDTLS := func(ctx context.Context, addr string, dtlsCfg *dtls.Config, opts ...udp.Option) (*coap.ClientCloseHandler, error) {
		opts = append(opts, udpDialOpts...)
		return coap.DialUDPSecure(ctx, addr, dtlsCfg, opts...)
	}
	dialTCP := func(ctx context.Context, addr string, opts ...tcp.Option) (*coap.ClientCloseHandler, error) {
		opts = append(opts, tcpDialOpts...)
		return coap.DialTCP(ctx, addr, opts...)
	}
	dialUDP := func(ctx context.Context, addr string, opts ...udp.Option) (*coap.ClientCloseHandler, error) {
		opts = append(opts, udpDialOpts...)
		return coap.DialUDP(ctx, addr, opts...)
	}

	opts := []core.OptionFunc{
		core.WithDialDTLS(dialDTLS),
		core.WithDialTLS(dialTLS),
		core.WithDialTCP(dialTCP),
		core.WithDialUDP(dialUDP),
		core.WithLogger(logger),
	}

	deviceOwner, err := NewDeviceOwnerFromConfig(cfg, dialTLS, dialDTLS, app)
	if err != nil {
		return nil, err
	}
	return NewClient(app, deviceOwner, cacheExpiration, observerPollingInterval, opts...)
}

// NewClient constructs a new local client.
func NewClient(
	app ApplicationCallback,
	deviceOwner DeviceOwner,
	cacheExpiration time.Duration,
	observerPollingInterval time.Duration,
	opt ...core.OptionFunc,
) (*Client, error) {
	if app == nil {
		return nil, fmt.Errorf("missing application callback")
	}
	if deviceOwner == nil {
		return nil, fmt.Errorf("missing device owner callback")
	}
	var coreCfg core.Config
	for _, o := range opt {
		coreCfg = o(coreCfg)
	}

	if coreCfg.Logger == nil {
		coreCfg.Logger = core.NewNilLogger()
	}
	tls := core.TLSConfig{
		GetCertificate:            deviceOwner.GetIdentityCertificate,
		GetCertificateAuthorities: deviceOwner.GetIdentityCACerts,
	}
	opt = append(
		[]core.OptionFunc{
			core.WithTLS(&tls),
			core.WithLogger(coreCfg.Logger),
		},
		opt...,
	)
	oc := core.NewClient(opt...)
	pollInterval := time.Second * 10
	if cacheExpiration/2 > pollInterval {
		pollInterval = cacheExpiration / 2
	}
	client := Client{
		client:                  oc,
		app:                     app,
		deviceCache:             NewDeviceCache(cacheExpiration, pollInterval, coreCfg.Logger),
		observeResourceCache:    coapSync.NewMap[string, *observationsHandler](),
		deviceOwner:             deviceOwner,
		subscriptions:           make(map[string]subscription),
		observerPollingInterval: observerPollingInterval,
		logger:                  coreCfg.Logger,
	}
	return &client, nil
}

type ownFunc = func(ctx context.Context, deviceID string, otmClient []otm.Client, discoveryConfiguration core.DiscoveryConfiguration, opts ...core.OwnOption) (string, error)

type DeviceOwner interface {
	Initialization(ctx context.Context) error
	OwnDevice(ctx context.Context, deviceID string, otmTypes []OTMType, discoveryConfiguration core.DiscoveryConfiguration, own ownFunc, opts ...core.OwnOption) (string, error)

	GetIdentityCertificate() (tls.Certificate, error)
	GetIdentityCACerts() ([]*x509.Certificate, error)
}

// Client uses the underlying OCF local client.
type Client struct {
	app    ApplicationCallback
	client *core.Client

	deviceCache *DeviceCache

	observeResourceCache    *coapSync.Map[string, *observationsHandler]
	observerPollingInterval time.Duration

	deviceOwner DeviceOwner

	subscriptionsLock sync.Mutex
	subscriptions     map[string]subscription

	disableUDPEndpoints bool
	logger              core.Logger
}

func (c *Client) popSubscriptions() map[string]subscription {
	c.subscriptionsLock.Lock()
	defer c.subscriptionsLock.Unlock()
	s := c.subscriptions
	c.subscriptions = make(map[string]subscription)
	return s
}

func (c *Client) popSubscription(id string) (subscription, error) {
	c.subscriptionsLock.Lock()
	defer c.subscriptionsLock.Unlock()
	v, ok := c.subscriptions[id]
	if !ok {
		return nil, fmt.Errorf("cannot find observation %v", id)
	}
	delete(c.subscriptions, id)
	return v, nil
}

func (c *Client) insertSubscription(id string, s subscription) {
	c.subscriptionsLock.Lock()
	defer c.subscriptionsLock.Unlock()
	c.subscriptions[id] = s
}

func (c *Client) CoreClient() *core.Client {
	return c.client
}

// Close clears all connections and spawned goroutines by client.
func (c *Client) Close(ctx context.Context) error {
	for _, s := range c.popSubscriptions() {
		s.Cancel()
	}
	return c.deviceCache.Close(ctx)
}

func NewDeviceOwnerFromConfig(cfg *Config, dialTLS core.DialTLS, dialDTLS core.DialDTLS, app ApplicationCallback) (DeviceOwner, error) {
	if cfg.DeviceOwnershipSDK != nil {
		c, err := newDeviceOwnershipSDKFromConfig(app, dialTLS, dialDTLS, cfg.DeviceOwnershipSDK)
		if err != nil {
			return nil, fmt.Errorf("cannot create sdk signers: %w", err)
		}
		return c, nil
	}
	if cfg.DeviceOwnershipBackend != nil {
		c, err := newDeviceOwnershipBackendFromConfig(app, dialTLS, dialDTLS, cfg.DeviceOwnershipBackend)
		if err != nil {
			return nil, fmt.Errorf("cannot create server signers: %w", err)
		}
		return c, nil
	}
	return newDeviceOwnershipNone(), nil
}
