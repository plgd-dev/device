package local

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/net"
	"github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/schema"
)

type Device struct {
	schema.DeviceLinks
	tlsConfig     *TLSConfig
	multicastConn []*gocoap.MulticastClientConn

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

func NewDevice(links schema.DeviceLinks, conn *gocoap.ClientConn, multicastConn []*gocoap.MulticastClientConn, tlsConfig *TLSConfig) *Device {
	pool := make(map[string]*coap.Client)

	addr, err := net.Parse("coap://", conn.RemoteAddr())
	if err == nil {
		pool[addr.URL()] = coap.NewClient(conn)
	}

	return &Device{
		DeviceLinks:   links,
		tlsConfig:     tlsConfig,
		conn:          pool,
		multicastConn: multicastConn,
		observations:  &sync.Map{},
	}
}

// Close closes open connections to the device.
func (d *Device) Close(ctx context.Context) error {
	var errors []error
	err := d.stopObservations(ctx)
	if err != nil {
		errors = append(errors, err)
	}

	conns := make([]*coap.Client, 0, 4)
	d.lock.Lock()
	for key, conn := range d.conn {
		conns = append(conns, conn)
		delete(d.conn, key)
	}
	d.lock.Unlock()
	for _, conn := range conns {
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

func DialTCPSecure(ctx context.Context, addr string, closeSession func(error), cert tls.Certificate, cas []*x509.Certificate, verifyPeerCertificate func(verifyPeerCertificate *x509.Certificate) error) (*gocoap.ClientConn, error) {
	caPool := x509.NewCertPool()
	for _, ca := range cas {
		caPool.AddCert(ca)
	}

	tlsConfig := tls.Config{
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
		TLSConfig: &tlsConfig,
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

func (d *Device) connectToEndpoint(ctx context.Context, endpoint schema.Endpoint) (*coap.Client, error) {
	const errMsg = "cannot connect to %v: %v"
	addr, err := endpoint.GetAddr()
	if err != nil {
		return nil, err
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	conn, ok := d.conn[addr.URL()]
	if ok {
		return conn, nil
	}
	closeSession := func(error) {
		d.lock.Lock()
		defer d.lock.Unlock()
		delete(d.conn, addr.URL())
	}
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
		cert, err := d.tlsConfig.GetCertificate()
		if err != nil {
			return nil, fmt.Errorf(errMsg, addr.URL(), err)
		}
		cas, err := d.tlsConfig.GetCertificateAuthorities()
		if err != nil {
			return nil, fmt.Errorf(errMsg, addr.URL(), err)
		}
		c, err = DialTCPSecure(ctx, addr.String(), closeSession, cert, cas, coap.VerifyIndetityCertificate)
		if err != nil {
			return nil, fmt.Errorf(errMsg, addr.URL(), err)
		}
	default:
		return nil, fmt.Errorf(errMsg, addr.URL(), fmt.Errorf("unknown scheme :%v", addr.GetScheme()))
	}
	conn = coap.NewClient(c)
	d.conn[addr.URL()] = conn
	return conn, nil

}

// connect gets or creates a connection based on the resource link
func (d *Device) connect(ctx context.Context, href string) (*coap.Client, error) {
	link, ok := d.GetResourceLink(href)
	if !ok {
		return nil, fmt.Errorf("cannot get resource link for: %v: not found", href)
	}
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

func (d *Device) DeviceID() string                        { return d.ID }
func (d *Device) GetResourceLinks() []schema.ResourceLink { return d.Links }
func (d *Device) GetDeviceLinks() schema.DeviceLinks      { return d.DeviceLinks }
