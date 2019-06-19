package resource

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"
	"time"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/net"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/kit/sync"
	"github.com/go-ocf/sdk/local/resource/link"
	"github.com/go-ocf/sdk/schema"
	"github.com/gofrs/uuid"
)

// TCPClientFactory maintains the shared link cache and connection pool.
type TCPClientFactory struct {
	linkCache *link.Cache
	pool      *sync.Pool
}

// NewTCPClientFactory creates the client factory.
func NewTCPClientFactory(tlsConfig TLSConfig, linkCache *link.Cache) *TCPClientFactory {
	tcpPool := sync.NewPool()
	tcpPool.SetFactory(createTCPConnection(tlsConfig, tcpPool))
	return &TCPClientFactory{linkCache: linkCache, pool: tcpPool}
}

func (f *TCPClientFactory) GetLinks() (r []schema.ResourceLink) {
	for _, l := range f.linkCache.Items() {
		r = append(r, l)
	}
	return
}

// NewClient populates the link cache and creates the client
// that uses the shared link cache and connection pool.
func (f *TCPClientFactory) NewClient(
	_ *gocoap.ClientConn,
	links schema.DeviceLinks,
) (*Client, error) {
	f.linkCache.Update(links.ID, links.Links...)
	return f.NewClientFromCache()
}

// NewClientFromCache creates the client
// that uses the shared link cache and connection pool.
func (f *TCPClientFactory) NewClientFromCache() (*Client, error) {
	c := Client{
		linkCache: f.linkCache,
		pool:      f.pool,
		getAddr:   getTCPAddr,
	}
	return &c, nil
}

func (f *TCPClientFactory) CloseConnections(links schema.DeviceLinks) {
	closeConnections(f.pool, links)
}

func getTCPAddr(r schema.ResourceLink) (net.Addr, error) {
	addr, err := r.GetTCPSecureAddr()
	if err != nil {
		return r.GetTCPAddr()
	}
	return addr, err
}

func VerifyIndetityCertificate(cert *x509.Certificate) error {
	// verify EKU manually
	ekuHasClient := false
	ekuHasServer := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageClientAuth {
			ekuHasClient = true
		}
		if eku == x509.ExtKeyUsageServerAuth {
			ekuHasServer = true
		}
	}
	if !ekuHasClient {
		return fmt.Errorf("not contains ExtKeyUsageClientAuth")
	}
	if !ekuHasServer {
		return fmt.Errorf("not contains ExtKeyUsageServerAuth")
	}
	ekuHasOcfId := false
	for _, eku := range cert.UnknownExtKeyUsage {
		if eku.Equal(kitNetCoap.ExtendedKeyUsage_IDENTITY_CERTIFICATE) {
			ekuHasOcfId = true
			break
		}
	}
	if !ekuHasOcfId {
		return fmt.Errorf("not contains ExtKeyUsage with OCF ID(1.3.6.1.4.1.44924.1.6")
	}
	cn := strings.Split(cert.Subject.CommonName, ":")
	if len(cn) != 2 {
		return fmt.Errorf("invalid subject common name: %v", cert.Subject.CommonName)
	}
	if strings.ToLower(cn[0]) != "uuid" {
		return fmt.Errorf("invalid subject common name %v: 'uuid' - not found", cert.Subject.CommonName)
	}
	_, err := uuid.FromString(cn[1])
	if err != nil {
		return fmt.Errorf("invalid subject common name %v: %v", cert.Subject.CommonName, err)
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

func createTCPConnection(tlsConfig TLSConfig, p *sync.Pool) sync.PoolFunc {
	return func(ctx context.Context, urlraw string) (interface{}, error) {
		closeSession := func(error) { p.Delete(urlraw) }
		const errMsg = "cannot create tcp connection to %v: %v"
		url, err := url.Parse(urlraw)
		if err != nil {
			return nil, fmt.Errorf(errMsg, urlraw, err)
		}
		addr, err := net.ParseURL(url)
		if err != nil {
			return nil, fmt.Errorf(errMsg, urlraw, err)
		}
		scheme := addr.GetScheme()
		switch scheme {
		case schema.TCPScheme:
			conn, err := DialTCP(ctx, addr.String(), closeSession)
			if err != nil {
				return nil, fmt.Errorf(errMsg, urlraw, err)
			}
			return conn, nil
		case schema.TCPSecureScheme:
			cert, err := tlsConfig.GetCertificate()
			if err != nil {
				return nil, fmt.Errorf(errMsg, urlraw, err)
			}
			cas, err := tlsConfig.GetCertificateAuthorities()
			if err != nil {
				return nil, fmt.Errorf(errMsg, urlraw, err)
			}
			conn, err := DialTCPSecure(ctx, addr.String(), closeSession, cert, cas, VerifyIndetityCertificate)
			if err != nil {
				return nil, fmt.Errorf(errMsg, urlraw, err)
			}
			return conn, nil
		default:
			return nil, fmt.Errorf(errMsg, urlraw, fmt.Errorf("unknown scheme :%v", scheme))
		}
	}
}
