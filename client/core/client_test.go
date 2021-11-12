package core_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/stretchr/testify/require"

	ocf "github.com/plgd-dev/device/client/core"
	justworks "github.com/plgd-dev/device/client/core/otm/just-works"
	"github.com/plgd-dev/device/client/core/otm/manufacturer"
	pkgError "github.com/plgd-dev/device/pkg/error"
	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/test"
	"github.com/plgd-dev/kit/v2/security"
)

var DevsimNetHost = "devsim-net-bridge-" + test.MustGetHostname()

type Client struct {
	*ocf.Client
	mfgOtm       *manufacturer.Client
	justWorksOtm *justworks.Client

	DeviceID string
	*ocf.Device
	DeviceLinks schema.ResourceLinks
}

func NewTestSecureClient() (*Client, error) {
	identityCert := test.GenerateIdentityCert(CertIdentity)
	return NewTestSecureClientWithCert(identityCert, false, false)
}

func NewTestSecureClientWithTLS(disableDTLS, disableTCPTLS bool) (*Client, error) {
	identityCert := test.GenerateIdentityCert(CertIdentity)
	return NewTestSecureClientWithCert(identityCert, disableDTLS, disableTCPTLS)
}

func NewTestSecureClientWithCert(cert tls.Certificate, disableDTLS, disableTCPTLS bool) (*Client, error) {
	mfgCert, err := tls.X509KeyPair(test.MfgCert, test.MfgKey)
	if err != nil {
		return nil, err
	}

	mfgCa, err := security.ParseX509FromPEM(test.RootCACrt)
	if err != nil {
		return nil, err
	}

	identityIntermediateCA, err := security.ParseX509FromPEM(test.IdentityIntermediateCA)
	if err != nil {
		return nil, err
	}
	identityIntermediateCAKeyBlock, _ := pem.Decode(test.IdentityIntermediateCAKey)
	if identityIntermediateCAKeyBlock == nil {
		return nil, fmt.Errorf("identityIntermediateCAKeyBlock is empty")
	}
	identityIntermediateCAKey, err := x509.ParseECPrivateKey(identityIntermediateCAKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 86400)
	signer := test.NewIdentityCertificateSigner(identityIntermediateCA, identityIntermediateCAKey, notBefore, notAfter)

	var manOpts []manufacturer.OptionFunc
	if disableDTLS {
		manOpts = append(manOpts, manufacturer.WithDialDTLS(func(ctx context.Context, addr string, dtlsCfg *dtls.Config, opts ...coap.DialOptionFunc) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}
	if disableTCPTLS {
		manOpts = append(manOpts, manufacturer.WithDialTLS(func(ctx context.Context, addr string, tlsCfg *tls.Config, opts ...coap.DialOptionFunc) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}

	mfgOtm := manufacturer.NewClient(mfgCert, mfgCa, signer.Sign, manOpts...)
	justWorksOtm := justworks.NewClient(signer.Sign)

	var opts []ocf.OptionFunc
	if disableDTLS {
		opts = append(opts, ocf.WithDialDTLS(func(ctx context.Context, addr string, dtlsCfg *dtls.Config, opts ...coap.DialOptionFunc) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}
	if disableTCPTLS {
		opts = append(opts, ocf.WithDialTLS(func(ctx context.Context, addr string, tlsCfg *tls.Config, opts ...coap.DialOptionFunc) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}
	opts = append(opts, ocf.WithTLS(&ocf.TLSConfig{
		GetCertificate: func() (tls.Certificate, error) {
			return cert, nil
		},
		GetCertificateAuthorities: func() ([]*x509.Certificate, error) {
			return identityIntermediateCA, nil
		}}),
	)

	c := ocf.NewClient(opts...)

	return &Client{Client: c, mfgOtm: mfgOtm, justWorksOtm: justWorksOtm}, nil
}

func (c *Client) SetUpTestDevice(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.DevsimNetBridge)

	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	device, err := c.GetDeviceByMulticast(timeout, secureDeviceID, ocf.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	eps := device.GetEndpoints()
	links, err := device.GetResourceLinks(timeout, eps)
	require.NoError(t, err)
	err = device.Own(timeout, links, c.mfgOtm)
	require.NoError(t, err)
	links, err = device.GetResourceLinks(timeout, eps)
	require.NoError(t, err)
	c.Device = device
	c.DeviceID = secureDeviceID
	c.DeviceLinks = links
}

func (c *Client) Close() {
	if c.DeviceID == "" {
		return
	}
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := c.Disown(timeout, c.DeviceLinks)
	if err != nil {
		panic(err)
	}
	c.Device.Close(timeout)
}

var (
	CertIdentity = "00000000-0000-0000-0000-000000000001"
)
