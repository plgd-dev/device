package local

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"
)

type deviceOwnershipClient struct {
	*CoapClient
	ownership schema.Doxm
}

func (c *deviceOwnershipClient) GetOwnership() schema.Doxm {
	return c.ownership
}

type deviceOwnershipHandler struct {
	deviceID string
	cancel   context.CancelFunc

	client *deviceOwnershipClient
	lock   sync.Mutex
	err    error
}

func newDeviceOwnershipHandler(deviceID string, cancel context.CancelFunc) *deviceOwnershipHandler {
	return &deviceOwnershipHandler{deviceID: deviceID, cancel: cancel}
}

func (h *deviceOwnershipHandler) Handle(ctx context.Context, clientConn *gocoap.ClientConn, ownership schema.Doxm) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if ownership.DeviceId == h.deviceID {
		h.client = &deviceOwnershipClient{CoapClient: NewCoapClient(clientConn), ownership: ownership}
		h.cancel()
	}
}

func (h *deviceOwnershipHandler) Error(err error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.err = err
}

func (h *deviceOwnershipHandler) Err() error {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.err
}

func (h *deviceOwnershipHandler) Client() *deviceOwnershipClient {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.client
}

func (c *Client) ownDeviceFindClient(ctx context.Context, deviceID string, discoveryTimeout time.Duration, status resource.DiscoverOwnershipStatus) (*deviceOwnershipClient, error) {
	ctxOwn, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()
	h := newDeviceOwnershipHandler(deviceID, cancel)

	err := c.GetDeviceOwnership(ctxOwn, status, h)
	client := h.Client()

	if client != nil {
		return client, nil
	}
	if err != nil {
		return nil, err
	}

	return nil, h.Err()
}

func mfgCertDial(addr string, cert tls.Certificate, ca []*x509.Certificate) (*gocoap.ClientConn, error) {
	caPool := x509.NewCertPool()
	for _, c := range ca {
		caPool.AddCert(c)
	}

	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
		//RootCAs:            caPool,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			for _, rawCert := range rawCerts {
				cert, err := x509.ParseCertificate(rawCert)
				if err != nil {
					return err
				}

				_, err = cert.Verify(x509.VerifyOptions{
					//Intermediates: intermediates,
					Roots:       caPool,
					CurrentTime: time.Now(),
					KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
				})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	return gocoap.DialWithTLS("tcp", addr, &tlsConfig)
}

type OTMClient interface {
	Type() schema.OwnerTransferMethod
	Dial(addr string) (*CoapClient, error)
}

type ManufacturerOTMClient struct {
	manufacturerCertificate tls.Certificate
	manufacturerCA          []*x509.Certificate
}

func NewManufacturerOTMClient(manufacturerCertificate tls.Certificate, manufacturerCA []*x509.Certificate) *ManufacturerOTMClient {
	return &ManufacturerOTMClient{
		manufacturerCertificate: manufacturerCertificate,
		manufacturerCA:          manufacturerCA,
	}
}

func (*ManufacturerOTMClient) Type() schema.OwnerTransferMethod {
	return schema.ManufacturerCertificate
}

func (otmc *ManufacturerOTMClient) Dial(addr string) (*CoapClient, error) {
	caPool := x509.NewCertPool()
	for _, c := range otmc.manufacturerCA {
		caPool.AddCert(c)
	}

	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{otmc.manufacturerCertificate},
		//RootCAs:            caPool,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			for _, rawCert := range rawCerts {
				cert, err := x509.ParseCertificate(rawCert)
				if err != nil {
					return err
				}

				_, err = cert.Verify(x509.VerifyOptions{
					//Intermediates: intermediates,
					Roots:       caPool,
					CurrentTime: time.Now(),
					KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
				})
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	coapConn, err := gocoap.DialWithTLS("tcp", addr, &tlsConfig)
	if err != nil {
		return nil, err
	}
	return NewCoapClient(coapConn), nil
}

// OwnDevice set ownership of device
func (c *Client) OwnDevice(
	ctx context.Context,
	deviceID string,
	otmClient OTMClient,
	discoveryTimeout time.Duration,
) error {
	const errMsg = "cannot own device %v: %v"

	client, err := c.ownDeviceFindClient(ctx, deviceID, discoveryTimeout, resource.DiscoverAllDevices)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}
	ownership := client.GetOwnership()
	if ownership.Owned {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("device already owned by %v", ownership.DeviceOwner))
	}

	var supportOtm bool
	for _, s := range ownership.SupportedOwnerTransferMethods {
		if s == otmClient.Type() {
			supportOtm = true
		}
		break
	}
	if !supportOtm {
		return fmt.Errorf(errMsg, deviceID, fmt.Sprintf("ownership transfer method '%v' is unsupported, supported are: %v", otmClient.Type(), ownership.SupportedOwnerTransferMethods))
	}

	req := schema.DoxmUpdate{
		SelectOwnerTransferMethod: otmClient.Type(),
	}

	/*doxm doesn't send any content for select OTM*/
	err = client.UpdateResourceCBOR(ctx, "/oic/sec/doxm", "", req, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	deviceLink, err := client.GetDeviceLinks(ctx, deviceID)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}
	var resourceLink schema.ResourceLink
	for _, link := range deviceLink.Links {
		if link.HasType("oic.r.doxm") {
			resourceLink = link
			break
		}
	}
	tcpTLSAddr, err := resourceLink.GetTCPTLSAddr()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("secure endpoints not found"))
	}

	tlsClient, err := otmClient.Dial(tcpTLSAddr.String())
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot create TLS connection: %v", err))
	}

	var provisionState schema.ProvisionState
	err = tlsClient.GetResourceCBOR(ctx, "/oic/sec/pstat", "", &provisionState)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get provision state %v", err))
	}

	fmt.Printf("provisionState %+v\n", provisionState)

	return nil
}
