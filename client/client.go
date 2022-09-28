package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/client/core/otm"
	"github.com/plgd-dev/device/pkg/net/coap"
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
func NewClientFromConfig(cfg *Config, app ApplicationCallback, errors func(error)) (*Client, error) {
	var cacheExpiration time.Duration
	switch {
	case cfg.DeviceCacheExpirationSeconds < 0:
		cacheExpiration = time.Microsecond
	case cfg.DeviceCacheExpirationSeconds == 0:
		cacheExpiration = time.Second * 3600
	default:
		cacheExpiration = time.Second * time.Duration(cfg.DeviceCacheExpirationSeconds)
	}

	observerPollingInterval := time.Second * 3
	if cfg.ObserverPollingIntervalSeconds > 0 {
		observerPollingInterval = time.Second * time.Duration(cfg.ObserverPollingIntervalSeconds)
	}

	tcpDialOpts := make([]tcp.Option, 0, 5)
	udpDialOpts := make([]udp.Option, 0, 5)

	errFn := func(error) {
		// ignore error
	}
	if errors != nil {
		errFn = errors
	}
	tcpDialOpts = append(tcpDialOpts, options.WithErrors(errFn))

	keepAliveConnectionTimeoutSeconds := time.Second * 60
	if cfg.KeepAliveConnectionTimeoutSeconds > 0 {
		keepAliveConnectionTimeoutSeconds = time.Second * time.Duration(cfg.KeepAliveConnectionTimeoutSeconds)
	}
	tcpDialOpts = append(tcpDialOpts, options.WithKeepAlive(3, keepAliveConnectionTimeoutSeconds/3, func(cc *tcpClient.Conn) {
		errFn(fmt.Errorf("keepalive failed for tcp: %v", cc.RemoteAddr()))
		cc.Close()
	}))
	udpDialOpts = append(udpDialOpts, options.WithKeepAlive(3, keepAliveConnectionTimeoutSeconds/3, func(cc *udpClient.Conn) {
		errFn(fmt.Errorf("keepalive failed for udp: %v", cc.RemoteAddr()))
		cc.Close()
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
	}

	deviceOwner, err := NewDeviceOwnerFromConfig(cfg, dialTLS, dialDTLS, app, errors)
	if err != nil {
		return nil, err
	}
	return NewClient(app, deviceOwner, cacheExpiration, observerPollingInterval, errors, opts...)
}

// NewClient constructs a new local client.
func NewClient(
	app ApplicationCallback,
	deviceOwner DeviceOwner,
	cacheExpiration time.Duration,
	observerPollingInterval time.Duration,
	errors func(error),
	opt ...core.OptionFunc,
) (*Client, error) {
	if app == nil {
		return nil, fmt.Errorf("missing application callback")
	}
	if deviceOwner == nil {
		return nil, fmt.Errorf("missing device owner callback")
	}
	tls := core.TLSConfig{
		GetCertificate:            deviceOwner.GetIdentityCertificate,
		GetCertificateAuthorities: deviceOwner.GetIdentityCACerts,
	}
	opt = append(
		[]core.OptionFunc{
			core.WithTLS(&tls),
		},
		opt...,
	)
	if errors != nil {
		opt = append(opt, core.WithErr(errors))
	}
	oc := core.NewClient(opt...)
	client := Client{
		client:                  oc,
		app:                     app,
		deviceCache:             NewRefDeviceCache(cacheExpiration, errors),
		observeResourceCache:    coapSync.NewMap[string, *observationsHandler](),
		deviceOwner:             deviceOwner,
		subscriptions:           make(map[string]subscription),
		observerPollingInterval: observerPollingInterval,
		errors:                  errors,
	}
	return &client, nil
}

type ownFunc = func(ctx context.Context, deviceID string, otmClient otm.Client, discoveryConfiguration core.DiscoveryConfiguration, opts ...core.OwnOption) (string, error)

type DeviceOwner interface {
	Initialization(ctx context.Context) error
	OwnDevice(ctx context.Context, deviceID string, otmType OTMType, discoveryConfiguration core.DiscoveryConfiguration, own ownFunc, opts ...core.OwnOption) (string, error)

	GetIdentityCertificate() (tls.Certificate, error)
	GetIdentityCACerts() ([]*x509.Certificate, error)
}

// Client uses the underlying OCF local client.
type Client struct {
	app    ApplicationCallback
	client *core.Client

	deviceCache *refDeviceCache

	observeResourceCache    *coapSync.Map[string, *observationsHandler]
	observerPollingInterval time.Duration

	deviceOwner DeviceOwner

	subscriptionsLock sync.Mutex
	subscriptions     map[string]subscription

	disableUDPEndpoints bool
	errors              func(error)
}

func (c *Client) popSubscriptions() map[string]subscription {
	c.subscriptionsLock.Lock()
	defer c.subscriptionsLock.Unlock()
	s := c.subscriptions
	c.subscriptions = make(map[string]subscription)
	return s
}

func (c *Client) popSubscription(ID string) (subscription, error) {
	c.subscriptionsLock.Lock()
	defer c.subscriptionsLock.Unlock()
	v, ok := c.subscriptions[ID]
	if !ok {
		return nil, fmt.Errorf("cannot find observation %v", ID)
	}
	delete(c.subscriptions, ID)
	return v, nil
}

func (c *Client) insertSubscription(ID string, s subscription) {
	c.subscriptionsLock.Lock()
	defer c.subscriptionsLock.Unlock()
	c.subscriptions[ID] = s
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

func NewDeviceOwnerFromConfig(cfg *Config, dialTLS core.DialTLS, dialDTLS core.DialDTLS, app ApplicationCallback, errors func(error)) (DeviceOwner, error) {
	if cfg.DeviceOwnershipSDK != nil {
		c, err := NewDeviceOwnershipSDKFromConfig(app, dialTLS, dialDTLS, cfg.DeviceOwnershipSDK)
		if err != nil {
			return nil, fmt.Errorf("cannot create sdk signers: %w", err)
		}
		return c, nil
	} else if cfg.DeviceOwnershipBackend != nil {
		c, err := NewDeviceOwnershipBackendFromConfig(app, dialTLS, dialDTLS, cfg.DeviceOwnershipBackend, errors)
		if err != nil {
			return nil, fmt.Errorf("cannot create server signers: %w", err)
		}
		return c, nil
	} else {
		return NewDeviceOwnershipNone(), nil
	}
}
