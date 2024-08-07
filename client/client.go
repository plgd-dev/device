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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/pion/dtls/v3"
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/client/core/otm"
	"github.com/plgd-dev/device/v2/internal/math"
	"github.com/plgd-dev/device/v2/pkg/log"
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
	DeviceCacheExpirationSeconds   uint64
	ObserverPollingIntervalSeconds uint64 // 0 means 3 seconds
	ObserverFailureThreshold       uint8  // 0 means 3

	KeepAliveConnectionTimeoutSeconds uint64 // 0 means keepalive is disabled
	MaxMessageSize                    uint32
	DisablePeerTCPSignalMessageCSMs   bool
	DefaultTransferDurationSeconds    uint64 // 0 means 15 seconds

	UseDeviceIDInQuery bool // if true, deviceID is used also in query. Set this option if you use bridged devices.

	// specify one of:
	DeviceOwnershipSDK     *DeviceOwnershipSDKConfig     `yaml:",omitempty"`
	DeviceOwnershipBackend *DeviceOwnershipBackendConfig `yaml:",omitempty"`
}

func toDuration(seconds uint64, def time.Duration) (time.Duration, error) {
	if seconds == 0 {
		return def, nil
	}
	const maxDurationSeconds uint64 = (1<<63 - 1) / uint64(time.Second)
	if seconds > maxDurationSeconds {
		return 0, errors.New("invalid value: interval overflows maximal duration")
	}
	return math.CastTo[time.Duration](seconds * uint64(time.Second)), nil
}

// NewClientFromConfig constructs a new local client from the proto configuration.
func NewClientFromConfig(cfg *Config, app ApplicationCallback, logger core.Logger) (*Client, error) {
	cacheExpiration, err := toDuration(cfg.DeviceCacheExpirationSeconds, 0)
	if err != nil {
		return nil, errors.New("invalid DeviceCacheExpirationSeconds value")
	}

	observerPollingInterval, err := toDuration(cfg.ObserverPollingIntervalSeconds, time.Second*3)
	if err != nil {
		return nil, errors.New("invalid ObserverPollingIntervalSeconds value")
	}

	tcpDialOpts := make([]tcp.Option, 0, 5)
	udpDialOpts := make([]udp.Option, 0, 5)

	if logger == nil {
		logger = log.NewNilLogger()
	}

	errFn := func(err error) {
		logger.Debug(err.Error())
	}

	tcpDialOpts = append(tcpDialOpts, options.WithErrors(errFn))
	udpDialOpts = append(udpDialOpts, options.WithErrors(errFn))

	keepAliveConnectionTimeout, err := toDuration(cfg.KeepAliveConnectionTimeoutSeconds, time.Second*60)
	if err != nil {
		return nil, errors.New("invalid KeepAliveConnectionTimeoutSeconds value")
	}
	tcpDialOpts = append(tcpDialOpts, options.WithKeepAlive(3, keepAliveConnectionTimeout/3, func(cc *tcpClient.Conn) {
		errFn(fmt.Errorf("keepalive failed for tcp %v", cc.RemoteAddr()))
		if errC := cc.Close(); errC != nil {
			errFn(fmt.Errorf("failed to close tcp connection %v: %w", cc.RemoteAddr(), errC))
		}
	}))
	udpDialOpts = append(udpDialOpts, options.WithKeepAlive(3, keepAliveConnectionTimeout/3, func(cc *udpClient.Conn) {
		errFn(fmt.Errorf("keepalive failed for udp %v", cc.RemoteAddr()))
		if errC := cc.Close(); errC != nil {
			errFn(fmt.Errorf("failed to close udp connection %v: %w", cc.RemoteAddr(), errC))
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

	defaultTransferDuration, err := toDuration(cfg.DefaultTransferDurationSeconds, time.Second*15)
	if err != nil {
		return nil, errors.New("invalid DefaultTransferDurationSeconds value")
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

	opts := []ClientOptionFunc{
		WithDialDTLS(dialDTLS),
		WithDialTLS(dialTLS),
		WithDialTCP(dialTCP),
		WithDialUDP(dialUDP),
		WithLogger(logger),
		WithObserverConfig(ObserverConfig{
			PollingInterval:  observerPollingInterval,
			FailureThreshold: cfg.ObserverFailureThreshold,
		}),
		WithCacheExpiration(cacheExpiration),
		WithUseDeviceIDInQuery(cfg.UseDeviceIDInQuery),
	}

	deviceOwner, err := NewDeviceOwnerFromConfig(cfg, dialTLS, dialDTLS, app)
	if err != nil {
		return nil, err
	}
	return NewClient(app, deviceOwner, opts...)
}

// ObserverConfig is a configuration of the devices observation.
type ObserverConfig struct {
	// PollingInterval is a time between two consecutive observations.
	PollingInterval time.Duration
	// FailureThreshold is a number of consecutive observation failures after which the device is marked as offline.
	FailureThreshold uint8
}

type ClientConfig struct {
	CoreOptions []core.OptionFunc
	// CacheExpiration is a time after which the device entry in cache is invalidated.
	CacheExpiration time.Duration
	// Observer is a configuration of the devices observation.
	Observer ObserverConfig
	// UseDeviceIDInQuery if true, deviceID is used also in query. Set this option if you use bridged devices.
	UseDeviceIDInQuery bool
}

type ClientOptionFunc func(ClientConfig) ClientConfig

// WithObserverConfig sets the observer config.
func WithObserverConfig(observerConfig ObserverConfig) ClientOptionFunc {
	return func(cfg ClientConfig) ClientConfig {
		if observerConfig.PollingInterval <= 0 {
			observerConfig.PollingInterval = 3 * time.Second
		}
		if observerConfig.FailureThreshold <= 0 {
			observerConfig.FailureThreshold = 3
		}
		cfg.Observer = observerConfig
		return cfg
	}
}

func WithCacheExpiration(cacheExpiration time.Duration) ClientOptionFunc {
	return func(cfg ClientConfig) ClientConfig {
		cfg.CacheExpiration = cacheExpiration
		return cfg
	}
}

func WithTLS(tlsConfig *core.TLSConfig) ClientOptionFunc {
	return func(cfg ClientConfig) ClientConfig {
		if tlsConfig != nil {
			cfg.CoreOptions = append(cfg.CoreOptions, core.WithTLS(tlsConfig))
		}
		return cfg
	}
}

func WithLogger(logger core.Logger) ClientOptionFunc {
	return func(cfg ClientConfig) ClientConfig {
		if logger != nil {
			cfg.CoreOptions = append(cfg.CoreOptions, core.WithLogger(logger))
		}
		return cfg
	}
}

func WithDialDTLS(dial core.DialDTLS) ClientOptionFunc {
	return func(cfg ClientConfig) ClientConfig {
		if dial != nil {
			cfg.CoreOptions = append(cfg.CoreOptions, core.WithDialDTLS(dial))
		}
		return cfg
	}
}

func WithDialTLS(dial core.DialTLS) ClientOptionFunc {
	return func(cfg ClientConfig) ClientConfig {
		if dial != nil {
			cfg.CoreOptions = append(cfg.CoreOptions, core.WithDialTLS(dial))
		}
		return cfg
	}
}

func WithDialTCP(dial core.DialTCP) ClientOptionFunc {
	return func(cfg ClientConfig) ClientConfig {
		if dial != nil {
			cfg.CoreOptions = append(cfg.CoreOptions, core.WithDialTCP(dial))
		}
		return cfg
	}
}

func WithDialUDP(dial core.DialUDP) ClientOptionFunc {
	return func(cfg ClientConfig) ClientConfig {
		if dial != nil {
			cfg.CoreOptions = append(cfg.CoreOptions, core.WithDialUDP(dial))
		}
		return cfg
	}
}

// WithUseDeviceIDInQuery sets the observer config.
func WithUseDeviceIDInQuery(useDeviceIDInQuery bool) ClientOptionFunc {
	return func(cfg ClientConfig) ClientConfig {
		cfg.UseDeviceIDInQuery = useDeviceIDInQuery
		return cfg
	}
}

// NewClient constructs a new local client.
func NewClient(
	app ApplicationCallback,
	deviceOwner DeviceOwner,
	opt ...ClientOptionFunc,
) (*Client, error) {
	if app == nil {
		return nil, errors.New("missing application callback")
	}
	if deviceOwner == nil {
		return nil, errors.New("missing device owner callback")
	}
	clientCfg := ClientConfig{
		CacheExpiration: time.Hour,
		Observer: ObserverConfig{
			PollingInterval:  time.Second * 3,
			FailureThreshold: 3,
		},
	}
	for _, o := range opt {
		clientCfg = o(clientCfg)
	}

	var coreCfg core.Config
	for _, o := range clientCfg.CoreOptions {
		coreCfg = o(coreCfg)
	}

	if coreCfg.Logger == nil {
		coreCfg.Logger = log.NewNilLogger()
	}
	tls := core.TLSConfig{
		GetCertificate:            deviceOwner.GetIdentityCertificate,
		GetCertificateAuthorities: deviceOwner.GetIdentityCACerts,
	}
	clientCfg.CoreOptions = append(
		[]core.OptionFunc{
			core.WithTLS(&tls),
			core.WithLogger(coreCfg.Logger),
		},
		clientCfg.CoreOptions...,
	)
	oc := core.NewClient(clientCfg.CoreOptions...)
	pollInterval := time.Second * 10
	if clientCfg.CacheExpiration/2 > pollInterval {
		pollInterval = clientCfg.CacheExpiration / 2
	}
	client := Client{
		client:               oc,
		app:                  app,
		deviceCache:          NewDeviceCache(clientCfg.CacheExpiration, pollInterval, coreCfg.Logger),
		observeResourceCache: coapSync.NewMap[string, *observationsHandler](),
		deviceOwner:          deviceOwner,
		subscriptions:        make(map[string]subscription),
		observerConfig:       clientCfg.Observer,
		logger:               coreCfg.Logger,
		useDeviceIDInQuery:   clientCfg.UseDeviceIDInQuery,
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

	observeResourceCache *coapSync.Map[string, *observationsHandler]
	observerConfig       ObserverConfig

	deviceOwner DeviceOwner

	subscriptionsLock sync.Mutex
	subscriptions     map[string]subscription

	disableUDPEndpoints bool
	logger              core.Logger

	useDeviceIDInQuery bool
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
