package local

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sort"
	"time"

	"github.com/go-ocf/kit/net/coap"

	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/sdk/schema"
)

// RetryFunc defines policy to repeat GetResource on error.
type RetryFunc = func() (when time.Time, err error)

// RetryFuncFactory defines factory of policy to repeat GetResource on error.
type RetryFuncFactory = func() RetryFunc

// ErrFunc to log errors in goroutines
type ErrFunc = func(err error)

// ResolveEndpointsFunc gets endpoints to resource, order is determined by position at array (0-highest)
type ResolveEndpointsFunc = func(ctx context.Context, href string, links schema.ResourceLinks) ([]schema.Endpoint, error)

// Client an OCF local client.
type Client struct {
	tlsConfig              *TLSConfig
	retryFuncFactory       RetryFuncFactory
	retrieveTimeout        time.Duration
	resolveEndpointsFunc   ResolveEndpointsFunc
	errFunc                ErrFunc
	dialOptions            []coap.DialOptionFunc
	discoveryConfiguration DiscoveryConfiguration
}

func checkTLSConfig(cfg *TLSConfig) *TLSConfig {
	if cfg == nil {
		cfg = new(TLSConfig)
	}
	if cfg.GetCertificate == nil {
		cfg.GetCertificate = func() (tls.Certificate, error) {
			return tls.Certificate{}, fmt.Errorf("not supported")
		}
	}
	if cfg.GetCertificateAuthorities == nil {
		cfg.GetCertificateAuthorities = func() ([]*x509.Certificate, error) {
			return nil, fmt.Errorf("not supported")
		}
	}
	return cfg
}

type config struct {
	tlsConfig              *TLSConfig
	retryFuncFactory       RetryFuncFactory
	retrieveTimeout        time.Duration
	errFunc                ErrFunc
	resolveEndpointsFunc   ResolveEndpointsFunc
	dialOptions            []coap.DialOptionFunc
	discoveryConfiguration DiscoveryConfiguration
}

type OptionFunc func(config) config

func WithTLS(tlsConfig *TLSConfig) OptionFunc {
	return func(cfg config) config {
		if tlsConfig != nil {
			cfg.tlsConfig = tlsConfig
		}
		return cfg
	}
}

// DiscoveryConfiguration setup discovery configuration
type DiscoveryConfiguration struct {
	MulticastHopLimit    int      // default: 2, min value: 1 - don't pass through router, max value: 255, https://tools.ietf.org/html/rfc2460#section-3
	MulticastAddressUDP4 []string // default: "[224.0.1.187:5683] (local.DiscoveryAddressUDP4), empty: don't use ipv4 multicast"
	MulticastAddressUDP6 []string // default: "[ff02::158]:5683", "[ff03::158]:5683", "[ff05::158]:5683]"] (local.DiscoveryAddressUDP6), empty: don't use ipv6 multicast"
}

// WithDiscoveryConfiguration override default DiscoveryConfiguration
func WithDiscoveryConfiguration(d DiscoveryConfiguration) OptionFunc {
	return func(cfg config) config {
		cfg.discoveryConfiguration = d
		return cfg
	}
}

func WithDialOptions(opts ...coap.DialOptionFunc) OptionFunc {
	return func(cfg config) config {
		if len(opts) > 0 {
			cfg.dialOptions = opts
		}
		return cfg
	}
}

func WithRetryPolicy(retryFuncFactory RetryFuncFactory, retrieveTimeout time.Duration) OptionFunc {
	return func(cfg config) config {
		if retryFuncFactory != nil {
			cfg.retryFuncFactory = retryFuncFactory
		}
		if retrieveTimeout > 0 {
			cfg.retrieveTimeout = retrieveTimeout
		}
		return cfg
	}
}

func WithErr(errFunc ErrFunc) OptionFunc {
	return func(cfg config) config {
		if errFunc != nil {
			cfg.errFunc = errFunc
		}
		return cfg
	}
}

func WithResolveEndpoints(resolveEndpointsFunc ResolveEndpointsFunc) OptionFunc {
	return func(cfg config) config {
		if resolveEndpointsFunc != nil {
			cfg.resolveEndpointsFunc = resolveEndpointsFunc
		}
		return cfg
	}
}

type sortEndpointsByScheme []schema.Endpoint

func (a sortEndpointsByScheme) Len() int      { return len(a) }
func (a sortEndpointsByScheme) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortEndpointsByScheme) Less(i, j int) bool {
	addr1, _ := a[i].GetAddr()
	addr2, _ := a[j].GetAddr()
	prio := map[schema.Scheme]int{
		schema.TCPSecureScheme: 4,
		schema.UDPSecureScheme: 3,
		schema.UDPScheme:       2,
		schema.TCPScheme:       1,
	}

	paddr1, _ := prio[schema.Scheme(addr1.GetScheme())]
	paddr2, _ := prio[schema.Scheme(addr2.GetScheme())]

	return paddr1 < paddr2
}

func sortEndpoints(endpoints []schema.Endpoint) []schema.Endpoint {
	var eps []schema.Endpoint
	for _, ep := range endpoints {
		eps = append(eps, ep)
	}
	v := sortEndpointsByScheme(eps)
	sort.Sort(v)
	return eps
}

func NewClient(opts ...OptionFunc) *Client {
	cfg := config{
		retryFuncFactory: func() func() (time.Time, error) {
			return func() (time.Time, error) { return time.Time{}, fmt.Errorf("no retries configured") }
		},
		retrieveTimeout: time.Second,
		errFunc: func(err error) {
			log.Error(err)
		},
		resolveEndpointsFunc: func(ctx context.Context, href string, links schema.ResourceLinks) ([]schema.Endpoint, error) {
			link, ok := links.GetResourceLink(href)
			if !ok {
				return nil, fmt.Errorf("cannot get resource link for: %v: not found", href)
			}
			if len(link.Endpoints) == 0 {
				deviceLink, ok := links.GetResourceLink("/oic/d")
				if !ok {
					return nil, fmt.Errorf("cannot get resource link for %v: empty endpoints", href)
				}
				return sortEndpoints(deviceLink.Endpoints), nil
			}
			return sortEndpoints(link.Endpoints), nil
		},
		dialOptions: []coap.DialOptionFunc{
			coap.WithDialDisablePeerTCPSignalMessageCSMs(),
		},
		discoveryConfiguration: DiscoveryConfiguration{
			MulticastHopLimit:    2,
			MulticastAddressUDP4: DiscoveryAddressUDP4,
			MulticastAddressUDP6: DiscoveryAddressUDP6,
		},
	}
	for _, o := range opts {
		cfg = o(cfg)
	}

	cfg.tlsConfig = checkTLSConfig(cfg.tlsConfig)
	return &Client{
		tlsConfig:              cfg.tlsConfig,
		retryFuncFactory:       cfg.retryFuncFactory,
		retrieveTimeout:        cfg.retrieveTimeout,
		errFunc:                cfg.errFunc,
		resolveEndpointsFunc:   cfg.resolveEndpointsFunc,
		dialOptions:            cfg.dialOptions,
		discoveryConfiguration: cfg.discoveryConfiguration,
	}
}
