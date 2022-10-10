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
	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/client/core/otm"
	justworks "github.com/plgd-dev/device/v2/client/core/otm/just-works"
	"github.com/plgd-dev/device/v2/client/core/otm/manufacturer"
	pkgError "github.com/plgd-dev/device/v2/pkg/error"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/test"
	"github.com/plgd-dev/go-coap/v3/tcp"
	"github.com/plgd-dev/go-coap/v3/udp"
	"github.com/plgd-dev/kit/v2/security"
	"github.com/stretchr/testify/require"
)

type Client struct {
	*core.Client
	mfgOtm       *manufacturer.Client
	justWorksOtm *justworks.Client

	DeviceID string
	*core.Device
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

func NewTestSigner() (core.CertificateSigner, error) {
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
	return test.NewIdentityCertificateSigner(identityIntermediateCA, identityIntermediateCAKey, notBefore, notAfter), nil
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

	var manOpts []manufacturer.OptionFunc
	if disableDTLS {
		manOpts = append(manOpts, manufacturer.WithDialDTLS(func(ctx context.Context, addr string, dtlsCfg *dtls.Config, opts ...udp.Option) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}
	if disableTCPTLS {
		manOpts = append(manOpts, manufacturer.WithDialTLS(func(ctx context.Context, addr string, tlsCfg *tls.Config, opts ...tcp.Option) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}

	mfgOtm := manufacturer.NewClient(mfgCert, mfgCa, manOpts...)
	justWorksOtm := justworks.NewClient()

	var opts []core.OptionFunc
	if disableDTLS {
		opts = append(opts, core.WithDialDTLS(func(ctx context.Context, addr string, dtlsCfg *dtls.Config, opts ...udp.Option) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}
	if disableTCPTLS {
		opts = append(opts, core.WithDialTLS(func(ctx context.Context, addr string, tlsCfg *tls.Config, opts ...tcp.Option) (*coap.ClientCloseHandler, error) {
			return nil, pkgError.NotSupported()
		}))
	}
	opts = append(opts, core.WithTLS(&core.TLSConfig{
		GetCertificate: func() (tls.Certificate, error) {
			return cert, nil
		},
		GetCertificateAuthorities: func() ([]*x509.Certificate, error) {
			return identityIntermediateCA, nil
		},
	}),
	)

	c := core.NewClient(opts...)

	return &Client{Client: c, mfgOtm: mfgOtm, justWorksOtm: justWorksOtm}, nil
}

func (c *Client) SetUpTestDevice(t *testing.T) {
	signer, err := NewTestSigner()
	require.NoError(t, err)

	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)

	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	device, err := c.GetDeviceByMulticast(timeout, secureDeviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	eps := device.GetEndpoints()
	links, err := device.GetResourceLinks(timeout, eps)
	require.NoError(t, err)
	err = device.Own(timeout, links, []otm.Client{c.mfgOtm}, core.WithSetupCertificates(signer.Sign))
	require.NoError(t, err)
	links, err = device.GetResourceLinks(timeout, eps)
	require.NoError(t, err)
	c.Device = device
	c.DeviceID = secureDeviceID
	c.DeviceLinks = links
}

func (c *Client) Close() error {
	if c.DeviceID == "" {
		return nil
	}
	timeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := c.Disown(timeout, c.DeviceLinks)
	if err != nil {
		return err
	}
	return c.Device.Close(timeout)
}

var CertIdentity = "00000000-0000-0000-0000-000000000001"
