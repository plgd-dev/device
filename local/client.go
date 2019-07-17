package local

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/go-ocf/kit/log"
	"github.com/go-ocf/sdk/schema"
)

// RetryFunc defines factor of policy to repeat GetResource on error.
type RetryFunc = func() func() (when time.Time, err error)

// ErrFunc to log errors in goroutines
type ErrFunc = func(err error)

// ResolveEndpointsFunc gets endpoints to resource, order is determined by position at array (0-highest)
type ResolveEndpointsFunc = func(ctx context.Context, href string, links schema.ResourceLinks) ([]schema.Endpoint, error)

// Client an OCF local client.
type Client struct {
	tlsConfig            *TLSConfig
	retryFunc            RetryFunc
	retrieveTimeout      time.Duration
	resolveEndpointsFunc ResolveEndpointsFunc
	errFunc              ErrFunc
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
	tlsConfig            *TLSConfig
	retryFunc            RetryFunc
	retrieveTimeout      time.Duration
	errFunc              ErrFunc
	resolveEndpointsFunc ResolveEndpointsFunc
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

func WithRetryPolicy(retryFunc RetryFunc, retrieveTimeout time.Duration) OptionFunc {
	return func(cfg config) config {
		if retryFunc != nil {
			cfg.retryFunc = retryFunc
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

func NewClient(opts ...OptionFunc) *Client {
	cfg := config{
		retryFunc: func() func() (time.Time, error) {
			return func() (time.Time, error) { return time.Time{}, fmt.Errorf("retry reach limit") }
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
			return link.Endpoints, nil
		},
	}
	for _, o := range opts {
		cfg = o(cfg)
	}

	cfg.tlsConfig = checkTLSConfig(cfg.tlsConfig)
	return &Client{tlsConfig: cfg.tlsConfig, retryFunc: cfg.retryFunc, retrieveTimeout: cfg.retrieveTimeout, errFunc: cfg.errFunc, resolveEndpointsFunc: cfg.resolveEndpointsFunc}
}
