package localEx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/go-ocf/kit/net/coap"

	ocf "github.com/go-ocf/sdk/local"
	cache "github.com/patrickmn/go-cache"
)

type ApplicationCallback = interface {
	GetRootCertificateAuthorities() ([]*x509.Certificate, error)
	GetManufacturerCertificateAuthorities() ([]*x509.Certificate, error)
	GetManufacturerCertificate() (tls.Certificate, error)
}

type OnAuthorizationRequestFunc = func(authCodeURL string)

type subscription = interface {
	Cancel()
	Wait()
}

type Config struct {
	DeviceCacheExpirationSeconds      int64
	KeepAliveConnectionTimeoutSeconds uint64 // 0 means keepalive is disabled
	ObserverPollingIntervalSeconds    uint64 // 0 means 3 seconds
	DisableDTLS                       bool
	DisablePeerTCPSignalMessageCSMs   bool
	DisableUDPEndpoints               bool

	// specify one of:
	DeviceOwnershipSDK     *DeviceOwnershipSDKConfig     `yaml:",omitempty"`
	DeviceOwnershipBackend *DeviceOwnershipBackendConfig `yaml:",omitempty"`
}

// NewClientFromConfig constructs a new local client from the proto configuration.
func NewClientFromConfig(cfg *Config, app ApplicationCallback, errors func(error)) (*Client, error) {
	var cacheExpiration time.Duration
	if cfg.DeviceCacheExpirationSeconds < 0 {
		cacheExpiration = cache.NoExpiration
	} else {
		cacheExpiration = time.Second * time.Duration(cfg.DeviceCacheExpirationSeconds)
	}

	observerPollingInterval := time.Second * 3
	if cfg.ObserverPollingIntervalSeconds > 0 {
		observerPollingInterval = time.Second * time.Duration(cfg.ObserverPollingIntervalSeconds)
	}

	opts := make([]ocf.OptionFunc, 0, 1)
	if cfg.KeepAliveConnectionTimeoutSeconds > 0 {
		opts = append(opts, ocf.WithDialOptions(
			coap.WithKeepAlive(time.Second*time.Duration(cfg.KeepAliveConnectionTimeoutSeconds)),
		))
	}
	if cfg.DisableDTLS {
		opts = append(opts, ocf.WithoutDTLS())
	}
	if cfg.DisablePeerTCPSignalMessageCSMs {
		opts = append(opts, ocf.WithDialOptions(
			coap.WithDialDisablePeerTCPSignalMessageCSMs(),
		))
	}

	deviceOwner, err := NewDeviceOwnerFromConfig(cfg, app, errors)
	if err != nil {
		return nil, err
	}
	return NewLocalClient(app, deviceOwner, cacheExpiration, observerPollingInterval, cfg.DisableUDPEndpoints, errors, opts...)
}

// NewLocalClient constructs a new local client.
func NewLocalClient(
	app ApplicationCallback,
	deviceOwner DeviceOwner,
	cacheExpiration time.Duration,
	observerPollingInterval time.Duration,
	disableUDPEndpoints bool,
	errors func(error),
	opt ...ocf.OptionFunc,
) (*Client, error) {
	if app == nil {
		return nil, fmt.Errorf("missing application callback")
	}
	if deviceOwner == nil {
		return nil, fmt.Errorf("missing device owner callback")
	}
	tls := ocf.TLSConfig{
		GetCertificate:            deviceOwner.GetIdentityCertificate,
		GetCertificateAuthorities: app.GetRootCertificateAuthorities,
	}
	opt = append(
		[]ocf.OptionFunc{
			ocf.WithTLS(&tls),
		},
		opt...,
	)
	oc := ocf.NewClient(opt...)
	client := Client{
		client:                  oc,
		app:                     app,
		deviceCache:             NewRefDeviceCache(cacheExpiration, errors),
		observeDeviceCache:      make(map[string]*refDevice),
		deviceOwner:             deviceOwner,
		subscriptions:           make(map[string]subscription),
		observerPollingInterval: observerPollingInterval,
		disableUDPEndpoints:     disableUDPEndpoints,
	}
	return &client, nil
}

type ownFunc = func(ctx context.Context, deviceID string, otmClient ocf.OTMClient, opts ...ocf.OwnOption) error

type DeviceOwner interface {
	Initialization(ctx context.Context) error
	OwnDevice(ctx context.Context, deviceID string, own ownFunc, opts ...ocf.OwnOption) error

	GetAccessTokenURL(ctx context.Context) (string, error)
	GetOnboardAuthorizationCodeURL(ctx context.Context, deviceID string) (string, error)
	GetIdentityCertificate() (tls.Certificate, error)
	Close(ctx context.Context) error
}

// Client uses the underlying OCF local client.
type Client struct {
	app    ApplicationCallback
	client *ocf.Client

	deviceCache *refDeviceCache

	observeDeviceCache      map[string]*refDevice
	observeDeviceCacheLock  sync.Mutex
	observerPollingInterval time.Duration

	deviceOwner         DeviceOwner
	grpcCertificate     tls.Certificate
	identityCertificate tls.Certificate
	rootCA              []*x509.Certificate

	subscriptionsLock sync.Mutex
	subscriptions     map[string]subscription

	disableUDPEndpoints bool
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

// Close clears all connections and spawned goroutines by client.
func (c *Client) Close(ctx context.Context) error {
	var errors []error
	for _, s := range c.popSubscriptions() {
		s.Cancel()
	}
	err := c.deviceCache.Close(ctx)
	if err != nil {
		errors = append(errors, err)
	}

	// observeDeviceCache will be removed by cleaned by close deviceCache

	return nil
}

func NewDeviceOwnerFromConfig(cfg *Config, app ApplicationCallback, errors func(error)) (DeviceOwner, error) {
	if cfg.DeviceOwnershipSDK != nil {
		c, err := NewDeviceOwnershipSDKFromConfig(app, cfg.DeviceOwnershipSDK, cfg.DisableDTLS)
		if err != nil {
			return nil, fmt.Errorf("cannot create sdk signers: %w", err)
		}
		return c, nil
	} else if cfg.DeviceOwnershipBackend != nil {
		c, err := NewDeviceOwnershipBackendFromConfig(app, cfg.DeviceOwnershipBackend, cfg.DisableDTLS, errors)
		if err != nil {
			return nil, fmt.Errorf("cannot create server signers: %w", err)
		}
		return c, nil
	} else {
		return NewDeviceOwnershipNone(), nil
	}
}
