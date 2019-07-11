package local

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/go-ocf/kit/log"
)

// RetryFunc defines policy to repeat GetResource on error.
type RetryFunc = func() (when time.Time, err error)

// ErrFunc to log errors in goroutines
type ErrFunc = func(err error)

// Client an OCF local client.
type Client struct {
	tlsConfig       *TLSConfig
	retryFunc       RetryFunc
	retrieveTimeout time.Duration
	errFunc         ErrFunc
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
	tlsConfig       *TLSConfig
	retryFunc       RetryFunc
	retrieveTimeout time.Duration
	errFunc         ErrFunc
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

func NewClient(opts ...OptionFunc) *Client {
	cfg := config{
		retryFunc: func() func() (when time.Time, err error) {
			i := new(int)
			return func() (time.Time, error) {
				if *i > 5 {
					return time.Time{}, fmt.Errorf("retry reach limit")
				}
				when := time.Now().Add(time.Millisecond * 100 * time.Duration(*i))
				*i++
				return when, nil
			}
		}(),
		retrieveTimeout: time.Second,
		errFunc: func(err error) {
			log.Error(err)
		},
	}
	for _, o := range opts {
		cfg = o(cfg)
	}

	cfg.tlsConfig = checkTLSConfig(cfg.tlsConfig)
	return &Client{tlsConfig: cfg.tlsConfig, retryFunc: cfg.retryFunc, retrieveTimeout: cfg.retrieveTimeout, errFunc: cfg.errFunc}
}
