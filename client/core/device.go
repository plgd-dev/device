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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	goNet "net"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"
	"github.com/pion/dtls/v3"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
	"github.com/plgd-dev/kit/v2/net"
	uberAtom "go.uber.org/atomic"
	"golang.org/x/sync/semaphore"
)

type DeviceConfiguration struct {
	DialDTLS   DialDTLS
	DialTLS    DialTLS
	DialUDP    DialUDP
	DialTCP    DialTCP
	Logger     Logger
	TLSConfig  *TLSConfig
	GetOwnerID func() (string, error)
}

type Device struct {
	deviceID  uberAtom.String
	foundByIP uberAtom.String
	cfg       DeviceConfiguration

	observations *coapSync.Map[string, *observation]
	private      struct {
		lock         sync.Mutex
		conn         map[string]*conn
		deviceTypes  []string
		getEndpoints func() schema.Endpoints
	}
}

func (d *Device) UpdateBy(v *Device) {
	if d == v {
		return
	}

	d.SetDeviceID(v.DeviceID())
	// foundByIP can be overwritten only when it is set.
	foundByIP := v.foundByIP.Load()
	if foundByIP != "" {
		d.foundByIP.Store(foundByIP)
	}
	v.private.lock.Lock()
	deviceTypes := v.private.deviceTypes
	getEndpoints := v.private.getEndpoints
	v.private.lock.Unlock()

	d.private.lock.Lock()
	defer d.private.lock.Unlock()
	d.private.deviceTypes = deviceTypes
	d.private.getEndpoints = getEndpoints
}

func (d *Device) SetEndpoints(getEndpoints func() schema.Endpoints) {
	d.private.lock.Lock()
	defer d.private.lock.Unlock()
	d.private.getEndpoints = getEndpoints
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
	d := &Device{
		cfg:          cfg,
		observations: coapSync.NewMap[string, *observation](),
	}
	d.private.deviceTypes = deviceTypes
	d.private.getEndpoints = getEndpoints
	d.private.conn = make(map[string]*conn)
	d.SetDeviceID(deviceID)
	return d
}

func (d *Device) popConnections() []*conn {
	conns := make([]*conn, 0, 4)
	d.private.lock.Lock()
	defer d.private.lock.Unlock()
	for key, conn := range d.private.conn {
		delete(d.private.conn, key)
		conns = append(conns, conn)
	}
	return conns
}

// Close closes open connections to the device.
func (d *Device) Close(ctx context.Context) error {
	var errs *multierror.Error
	if err := d.closeObservations(ctx); err != nil {
		errs = multierror.Append(errs, err)
	}

	for _, conn := range d.popConnections() {
		if errC := conn.Close(); errC != nil && !errors.Is(errC, goNet.ErrClosed) {
			errs = multierror.Append(errs, errC)
		}
		// wait for closing socket
		<-conn.Done()
	}

	if errs.ErrorOrNil() != nil {
		return MakeInternal(fmt.Errorf("cannot close device %v: %w", d.DeviceID(), errs))
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
		InsecureSkipVerify:    true, //nolint:gosec
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
	d.private.lock.Lock()
	defer d.private.lock.Unlock()
	c, ok := d.private.conn[addr]
	if !ok {
		c = &conn{
			mutex: semaphore.NewWeighted(1),
		}
		d.private.conn[addr] = c
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
	d.private.lock.Lock()
	defer d.private.lock.Unlock()
	// check if the connection is still in the map
	c, ok := d.private.conn[addr]
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
	// check if the dial was called and never failed
	if clientConn == nil && c.err == nil {
		return
	}
	// check if the underlying connection is same as the one we want to remove
	if clientConn != nil && c.get() == clientConn {
		delete(d.private.conn, addr)
	} else if c == cc && c.err != nil {
		// check if the wrapped connection is the same we are about to delete
		delete(d.private.conn, addr)
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
		cc.RegisterCloseHandler(func(error) {
			d.removeConn(addr.URL(), conn)
		})
	}
	return addr, cc, nil
}

func (d *Device) connectToEndpoints(ctx context.Context, endpoints schema.Endpoints) (net.Addr, *coap.ClientCloseHandler, error) {
	var errs *multierror.Error

	for _, endpoint := range endpoints {
		addr, conn, err := d.connectToEndpoint(ctx, endpoint)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}
		return addr, conn, nil
	}
	if errs.ErrorOrNil() != nil {
		return net.Addr{}, nil, errs
	}
	return net.Addr{}, nil, MakeInternal(errors.New("cannot connect to empty endpoints"))
}

func (d *Device) DeviceID() string {
	return d.deviceID.Load()
}

func (d *Device) SetDeviceID(deviceID string) {
	d.deviceID.Store(deviceID)
}

func (d *Device) FoundByIP() string {
	return d.foundByIP.Load()
}

func (d *Device) IsConnected() bool {
	d.private.lock.Lock()
	defer d.private.lock.Unlock()
	return len(d.private.conn) > 0
}

func (d *Device) setFoundByIP(foundByIP string) {
	d.foundByIP.Store(foundByIP)
}

func (d *Device) DeviceTypes() []string {
	d.private.lock.Lock()
	deviceTypes := d.private.deviceTypes
	d.private.lock.Unlock()
	return deviceTypes
}
