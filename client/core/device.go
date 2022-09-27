package core

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	goNet "net"
	"sync"
	"sync/atomic"

	"github.com/pion/dtls/v2"
	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/kit/v2/net"
	"golang.org/x/sync/semaphore"
)

type DeviceConfiguration struct {
	DialDTLS   DialDTLS
	DialTLS    DialTLS
	DialUDP    DialUDP
	DialTCP    DialTCP
	ErrFunc    ErrFunc
	TLSConfig  *TLSConfig
	GetOwnerID func() (string, error)
}

type Device struct {
	deviceID     string
	foundByIP    string
	deviceTypes  []string
	getEndpoints func() schema.Endpoints
	cfg          DeviceConfiguration

	conn         map[string]*conn
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

type conn struct {
	mutex *semaphore.Weighted
	c     atomic.Value // *coap.ClientCloseHandler
	err   error
}

func (c *conn) get() *coap.ClientCloseHandler {
	v := c.c.Load()
	if v == nil {
		return nil
	}
	if cc, ok := v.(*coap.ClientCloseHandler); ok {
		return cc
	}
	return nil
}

func (c *conn) Dial(ctx context.Context, dial func(ctx context.Context, addr net.Addr) (*coap.ClientCloseHandler, error), addr net.Addr) (*coap.ClientCloseHandler, bool, error) {
	err := c.mutex.Acquire(ctx, 1)
	if err != nil {
		return nil, false, err
	}
	defer c.mutex.Release(1)
	clientConn := c.get()
	if clientConn != nil && clientConn.Context().Err() == nil {
		return clientConn, true, nil
	}
	if c.err != nil {
		return nil, false, c.err
	}
	conn, err := dial(ctx, addr)
	if err != nil {
		c.err = err
		return nil, false, err
	}
	c.c.Store(conn)
	return conn, false, nil
}

func (c *conn) Close() error {
	clientConn := c.get()
	if clientConn != nil {
		return clientConn.Close()
	}
	return nil
}

func (c *conn) Done() <-chan struct{} {
	clientConn := c.get()
	if clientConn != nil {
		return clientConn.Done()
	}
	done := make(chan struct{})
	close(done)
	return done
}

func NewDevice(
	cfg DeviceConfiguration,
	deviceID string,
	deviceTypes []string,
	getEndpoints func() schema.Endpoints,
) *Device {
	return &Device{
		cfg:          cfg,
		deviceID:     deviceID,
		deviceTypes:  deviceTypes,
		observations: &sync.Map{},
		getEndpoints: getEndpoints,
		conn:         make(map[string]*conn),
	}
}

func (d *Device) popConnections() []*conn {
	conns := make([]*conn, 0, 4)
	d.lock.Lock()
	defer d.lock.Unlock()
	for key, conn := range d.conn {
		delete(d.conn, key)
		conns = append(conns, conn)
	}
	return conns
}

// Close closes open connections to the device.
func (d *Device) Close(ctx context.Context) error {
	var errs []error
	if err := d.stopObservations(ctx); err != nil {
		errs = append(errs, err)
	}

	for _, conn := range d.popConnections() {
		if errC := conn.Close(); errC != nil && !errors.Is(errC, goNet.ErrClosed) {
			errs = append(errs, errC)
		}
		// wait for closing socket
		<-conn.Done()
	}

	if len(errs) > 0 {
		return MakeInternal(fmt.Errorf("cannot close device %v: %v", d.DeviceID(), errs))
	}
	return nil
}

func (d *Device) dialTLS(ctx context.Context, addr string, tlsConfig *TLSConfig, verifyPeerCertificate func(verifyPeerCertificate *x509.Certificate) error) (*coap.ClientCloseHandler, error) {
	cert, err := tlsConfig.GetCertificate()
	if err != nil {
		return nil, err
	}
	cas, err := tlsConfig.GetCertificateAuthorities()
	if err != nil {
		return nil, err
	}
	rootCAs := x509.NewCertPool()
	for _, ca := range cas {
		rootCAs.AddCert(ca)
	}
	tlsCfg := tls.Config{
		InsecureSkipVerify:    true,
		ClientCAs:             rootCAs,
		Certificates:          []tls.Certificate{cert},
		VerifyPeerCertificate: coap.NewVerifyPeerCertificate(rootCAs, verifyPeerCertificate),
	}

	return d.cfg.DialTLS(ctx, addr, &tlsCfg)
}

func (d *Device) dialDTLS(ctx context.Context, addr string, tlsConfig *TLSConfig, verifyPeerCertificate func(verifyPeerCertificate *x509.Certificate) error) (*coap.ClientCloseHandler, error) {
	cert, err := tlsConfig.GetCertificate()
	if err != nil {
		return nil, err
	}
	cas, err := tlsConfig.GetCertificateAuthorities()
	if err != nil {
		return nil, err
	}
	rootCAs := x509.NewCertPool()
	for _, ca := range cas {
		rootCAs.AddCert(ca)
	}

	tlsCfg := dtls.Config{
		InsecureSkipVerify:    true,
		ClientCAs:             rootCAs,
		CipherSuites:          []dtls.CipherSuiteID{dtls.TLS_ECDHE_ECDSA_WITH_AES_128_CCM_8, dtls.TLS_ECDHE_ECDSA_WITH_AES_128_CCM},
		Certificates:          []tls.Certificate{cert},
		VerifyPeerCertificate: coap.NewVerifyPeerCertificate(rootCAs, verifyPeerCertificate),
	}
	return d.cfg.DialDTLS(ctx, addr, &tlsCfg)
}

func (d *Device) loadORCreate(addr string) (c *conn) {
	d.lock.Lock()
	defer d.lock.Unlock()
	c, ok := d.conn[addr]
	if !ok {
		c = &conn{
			mutex: semaphore.NewWeighted(1),
		}
		d.conn[addr] = c
	}
	return c
}

func (d *Device) dial(ctx context.Context, addr net.Addr) (*coap.ClientCloseHandler, error) {
	switch schema.Scheme(addr.GetScheme()) {
	case schema.UDPScheme:
		return d.cfg.DialUDP(ctx, addr.String())
	case schema.UDPSecureScheme:
		return d.dialDTLS(ctx, addr.String(), d.cfg.TLSConfig, coap.VerifyIdentityCertificate)
	case schema.TCPScheme:
		return d.cfg.DialTCP(ctx, addr.String())
	case schema.TCPSecureScheme:
		return d.dialTLS(ctx, addr.String(), d.cfg.TLSConfig, coap.VerifyIdentityCertificate)
	}
	return nil, fmt.Errorf("unknown scheme :%v", addr.GetScheme())
}

func (d *Device) removeConn(addr string, cc *conn) {
	d.lock.Lock()
	defer d.lock.Unlock()
	// check if the connection is still in the map
	c, ok := d.conn[addr]
	if !ok {
		return
	}
	// check if the connection is not used anyone
	locked := c.mutex.TryAcquire(1)
	if !locked {
		return
	}
	defer c.mutex.Release(1)
	clientConn := cc.get()
	// check if the dial was called
	if clientConn == nil {
		return
	}
	// check if the underlayer connection is same as the one we want to remove
	if c.get() == clientConn {
		delete(d.conn, addr)
	}
}

func (d *Device) connectToEndpoint(ctx context.Context, endpoint schema.Endpoint) (net.Addr, *coap.ClientCloseHandler, error) {
	const errMsg = "cannot connect to %v: %w"
	addr, err := endpoint.GetAddr()
	if err != nil {
		return net.Addr{}, nil, err
	}
	conn := d.loadORCreate(addr.URL())
	cc, loaded, err := conn.Dial(ctx, d.dial, addr)
	if err != nil {
		d.removeConn(addr.URL(), conn)
		return net.Addr{}, nil, MakeInternal(fmt.Errorf(errMsg, addr.URL(), err))
	}
	if !loaded {
		cc.RegisterCloseHandler(func(err error) {
			d.removeConn(addr.URL(), conn)
		})
	}
	return addr, cc, nil
}

func (d *Device) connectToEndpoints(ctx context.Context, endpoints schema.Endpoints) (net.Addr, *coap.ClientCloseHandler, error) {
	errors := make([]error, 0, 4)

	for _, endpoint := range endpoints {
		addr, conn, err := d.connectToEndpoint(ctx, endpoint)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		return addr, conn, nil
	}
	if len(errors) > 0 {
		return net.Addr{}, nil, fmt.Errorf("%v", errors)
	}
	return net.Addr{}, nil, MakeInternal(fmt.Errorf("cannot connect to empty endpoints"))
}

func (d *Device) DeviceID() string {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.deviceID
}

func (d *Device) setDeviceID(deviceID string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.deviceID = deviceID
}

func (d *Device) FoundByIP() string {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.foundByIP
}

func (d *Device) IsConnected() bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	return len(d.conn) > 0
}

func (d *Device) setFoundByIP(foundByIP string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	d.foundByIP = foundByIP
}

func (d *Device) DeviceTypes() []string {
	return d.deviceTypes
}
