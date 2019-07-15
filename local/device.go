package local

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/schema"
)

type Device struct {
	deviceID        string
	deviceTypes     []string
	links           schema.ResourceLinks
	tlsConfig       *TLSConfig
	retryFunc       RetryFunc
	retrieveTimeout time.Duration
	errFunc         ErrFunc

	conn         map[string]*coap.Client
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

func NewDevice(tlsConfig *TLSConfig, retryFunc RetryFunc, retrieveTimeout time.Duration, errFunc ErrFunc, deviceID string, deviceTypes []string, links schema.ResourceLinks) *Device {
	pool := make(map[string]*coap.Client)

	return &Device{
		deviceID:        deviceID,
		deviceTypes:     deviceTypes,
		links:           links,
		tlsConfig:       tlsConfig,
		retryFunc:       retryFunc,
		retrieveTimeout: retrieveTimeout,
		conn:            pool,
		errFunc:         errFunc,
		observations:    &sync.Map{},
	}
}

func (d *Device) popConnections() []*coap.Client {
	conns := make([]*coap.Client, 0, 4)
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

func DialTCPSecure(ctx context.Context, addr string, closeSession func(error), tlsConfig *TLSConfig, verifyPeerCertificate func(verifyPeerCertificate *x509.Certificate) error) (*gocoap.ClientConn, error) {

	cert, err := tlsConfig.GetCertificate()
	if err != nil {
		return nil, err
	}
	cas, err := tlsConfig.GetCertificateAuthorities()
	if err != nil {
		return nil, err
	}
	caPool := x509.NewCertPool()
	for _, ca := range cas {
		caPool.AddCert(ca)
	}

	tlsCfg := tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			for _, rawCert := range rawCerts {
				cert, err := x509.ParseCertificate(rawCert)
				if err != nil {
					return err
				}
				_, err = cert.Verify(x509.VerifyOptions{
					Roots:       caPool,
					CurrentTime: time.Now(),
					KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
				})
				if err != nil {
					return err
				}
				if verifyPeerCertificate(cert) != nil {
					return err
				}
			}
			return nil
		},
	}
	client := gocoap.Client{Net: "tcp-tls", NotifySessionEndFunc: closeSession,
		TLSConfig: &tlsCfg,
		// Iotivity 1.3 breaks with signal messages,
		// but Iotivity 2.0 requires them.
		DisableTCPSignalMessages: false,
	}
	return client.DialWithContext(ctx, addr)
}

func DialTCP(ctx context.Context, addr string, closeSession func(error)) (*gocoap.ClientConn, error) {
	client := gocoap.Client{Net: "tcp", NotifySessionEndFunc: closeSession,
		// Iotivity 1.3 breaks with signal messages,
		// but Iotivity 2.0 requires them.
		DisableTCPSignalMessages: false,
	}
	return client.DialWithContext(ctx, addr)
}

func (d *Device) getConn(addr string) *coap.Client {
	d.lock.Lock()
	defer d.lock.Unlock()
	conn, ok := d.conn[addr]
	if ok {
		return conn
	}
	return nil
}

type removeConnection struct {
	device *Device
	addr   string
	remove bool
}

func (c *removeConnection) removeSession(err error) {
	c.device.lock.Lock()
	defer c.device.lock.Unlock()
	if !c.remove {
		return
	}
	delete(c.device.conn, c.addr)
}

func (d *Device) connectToEndpoint(ctx context.Context, endpoint schema.Endpoint) (*coap.Client, error) {
	const errMsg = "cannot connect to %v: %v"
	addr, err := endpoint.GetAddr()
	if err != nil {
		return nil, err
	}

	conn := d.getConn(addr.URL())
	if conn != nil {
		return conn, nil
	}

	rem := &removeConnection{
		device: d,
		addr:   addr.URL(),
	}

	closeSession := rem.removeSession
	var c *gocoap.ClientConn
	switch addr.GetScheme() {
	case schema.UDPScheme:
		client := gocoap.Client{Net: "udp", NotifySessionEndFunc: closeSession}
		c, err = client.DialWithContext(ctx, addr.String())
		if err != nil {
			return nil, fmt.Errorf(errMsg, addr.URL(), err)
		}
	case schema.UDPSecureScheme:
		return nil, fmt.Errorf(errMsg, addr.URL(), "not supported")
	case schema.TCPScheme:
		c, err = DialTCP(ctx, addr.String(), closeSession)
		if err != nil {
			return nil, fmt.Errorf(errMsg, addr.URL(), err)
		}
	case schema.TCPSecureScheme:
		c, err = DialTCPSecure(ctx, addr.String(), closeSession, d.tlsConfig, coap.VerifyIndetityCertificate)
		if err != nil {
			return nil, fmt.Errorf(errMsg, addr.URL(), err)
		}
	default:
		return nil, fmt.Errorf(errMsg, addr.URL(), fmt.Errorf("unknown scheme :%v", addr.GetScheme()))
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	conn, ok := d.conn[addr.URL()]
	if ok {
		c.Close()
		return conn, nil
	}
	rem.remove = true
	conn = coap.NewClient(c)
	d.conn[addr.URL()] = conn
	return conn, nil

}

func (d *Device) connectToLink(ctx context.Context, link schema.ResourceLink) (*coap.Client, error) {
	errors := make([]error, 0, 4)
	for _, endpoint := range link.GetEndpoints() {
		conn, err := d.connectToEndpoint(ctx, endpoint)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		return conn, nil
	}
	return nil, fmt.Errorf("%v", errors)
}

// connect gets or creates a connection based on the resource link
func (d *Device) connect(ctx context.Context, href string) (*coap.Client, error) {
	links, err := d.GetResourceLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get resource links: %v", err)
	}

	link, ok := links.GetResourceLink(href)
	if !ok {
		return nil, fmt.Errorf("cannot get resource link for: %v: not found", href)
	}
	return d.connectToLink(ctx, link)
}

func (d *Device) DeviceID() string      { return d.deviceID }
func (d *Device) DeviceTypes() []string { return d.deviceTypes }

//func (d *Device) GetResourceLinks() []schema.ResourceLink { return d.Links }
//func (d *Device) GetDeviceLinks() schema.DeviceLinks      { return d.DeviceLinks }
