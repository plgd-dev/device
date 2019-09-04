package local

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/schema"
)

type Device struct {
	deviceID               string
	deviceTypes            []string
	links                  schema.ResourceLinks
	tlsConfig              *TLSConfig
	retryFuncFactory       RetryFuncFactory
	retrieveTimeout        time.Duration
	errFunc                ErrFunc
	resolveEndpointsFunc   ResolveEndpointsFunc
	dialOptions            []coap.DialOptionFunc
	discoveryConfiguration DiscoveryConfiguration

	conn         map[string]*coap.ClientCloseHandler
	observations *sync.Map
	lock         sync.Mutex
}

// GetCertificateFunc returns certificate for connection
type GetCertificateFunc func() (tls.Certificate, error)

// GetCertificateAuthoritiesFunc returns certificate authorities to verify peers
type GetCertificateAuthoritiesFunc func() ([]*x509.Certificate, error)

type TLSConfig struct {
	// User for communication with owned devices and cloud
	GetCertificate            GetCertificateFunc
	GetCertificateAuthorities GetCertificateAuthoritiesFunc
}

func NewDevice(
	tlsConfig *TLSConfig,
	retryFuncFactory RetryFuncFactory,
	retrieveTimeout time.Duration,
	errFunc ErrFunc,
	resolveEndpointsFunc ResolveEndpointsFunc,
	dialOptions []coap.DialOptionFunc,
	discoveryConfiguration DiscoveryConfiguration,
	deviceID string,
	deviceTypes []string,
	links schema.ResourceLinks,
) *Device {
	pool := make(map[string]*coap.ClientCloseHandler)

	return &Device{
		deviceID:               deviceID,
		deviceTypes:            deviceTypes,
		links:                  links,
		tlsConfig:              tlsConfig,
		retryFuncFactory:       retryFuncFactory,
		retrieveTimeout:        retrieveTimeout,
		discoveryConfiguration: discoveryConfiguration,
		conn:                 pool,
		errFunc:              errFunc,
		resolveEndpointsFunc: resolveEndpointsFunc,
		observations:         &sync.Map{},
		dialOptions:          dialOptions,
	}
}

func (d *Device) popConnections() []*coap.ClientCloseHandler {
	conns := make([]*coap.ClientCloseHandler, 0, 4)
	d.lock.Lock()
	defer d.lock.Unlock()
	for key, conn := range d.conn {
		conns = append(conns, conn)
		delete(d.conn, key)
	}
	return conns
}

// Close closes open connections to the device.
func (d *Device) Close(ctx context.Context) error {
	var errors []error
	err := d.stopObservations(ctx)
	if err != nil {
		errors = append(errors, err)
	}

	for _, conn := range d.popConnections() {
		err = conn.Close()
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("cannot close device %v: %v", d.DeviceID(), errors)
	}
	return nil
}

func DialTCPSecure(ctx context.Context, addr string, tlsConfig *TLSConfig, verifyPeerCertificate func(verifyPeerCertificate *x509.Certificate) error, dialOptions ...coap.DialOptionFunc) (*coap.ClientCloseHandler, error) {
	cert, err := tlsConfig.GetCertificate()
	if err != nil {
		return nil, err
	}
	cas, err := tlsConfig.GetCertificateAuthorities()
	if err != nil {
		return nil, err
	}
	return coap.DialTCPSecure(ctx, addr, cert, cas, verifyPeerCertificate, dialOptions...)
}

func (d *Device) getConn(addr string) (c *coap.ClientCloseHandler, ok bool) {
	d.lock.Lock()
	defer d.lock.Unlock()
	c, ok = d.conn[addr]
	return
}

func (d *Device) connectToEndpoint(ctx context.Context, endpoint schema.Endpoint) (*coap.ClientCloseHandler, error) {
	const errMsg = "cannot connect to %v: %v"
	addr, err := endpoint.GetAddr()
	if err != nil {
		return nil, err
	}

	conn, ok := d.getConn(addr.URL())
	if ok {
		return conn, nil
	}

	var c *coap.ClientCloseHandler
	switch schema.Scheme(addr.GetScheme()) {
	case schema.UDPScheme:
		c, err = coap.DialUDP(ctx, addr.String(), d.dialOptions...)
		if err != nil {
			return nil, fmt.Errorf(errMsg, addr.URL(), err)
		}
	case schema.UDPSecureScheme:
		return nil, fmt.Errorf(errMsg, addr.URL(), "not supported")
	case schema.TCPScheme:
		c, err = coap.DialTCP(ctx, addr.String(), d.dialOptions...)
		if err != nil {
			return nil, fmt.Errorf(errMsg, addr.URL(), err)
		}
	case schema.TCPSecureScheme:
		c, err = DialTCPSecure(ctx, addr.String(), d.tlsConfig, coap.VerifyIndetityCertificate, d.dialOptions...)
		if err != nil {
			return nil, fmt.Errorf(errMsg, addr.URL(), err)
		}
	default:
		return nil, fmt.Errorf(errMsg, addr.URL(), fmt.Errorf("unknown scheme :%v", addr.GetScheme()))
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	conn, ok = d.conn[addr.URL()]
	if ok {
		c.Close()
		return conn, nil
	}
	c.RegisterCloseHandler(func(error) {
		d.lock.Lock()
		defer d.lock.Unlock()
		delete(d.conn, addr.URL())
	})
	d.conn[addr.URL()] = c
	return c, nil
}

func (d *Device) connectToEndpoints(ctx context.Context, endpoints []schema.Endpoint) (*coap.ClientCloseHandler, error) {
	errors := make([]error, 0, 4)

	for _, endpoint := range endpoints {
		conn, err := d.connectToEndpoint(ctx, endpoint)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		return conn, nil
	}
	if len(errors) > 0 {
		return nil, fmt.Errorf("%v", errors)
	}
	return nil, fmt.Errorf("cannot connect to empty endpoints")
}

// connect gets or creates a connection based on the resource link
func (d *Device) connect(ctx context.Context, href string) (*coap.ClientCloseHandler, error) {
	links, err := d.GetResourceLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get resource links: %v", err)
	}

	endpoints, err := d.resolveEndpointsFunc(ctx, href, links)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve endpoints: %v", err)
	}
	return d.connectToEndpoints(ctx, endpoints)
}

func (d *Device) DeviceID() string      { return d.deviceID }
func (d *Device) DeviceTypes() []string { return d.deviceTypes }
