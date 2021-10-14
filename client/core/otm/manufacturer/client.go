package manufacturer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/pion/dtls/v2"
	kitNetCoap "github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/credential"
	"github.com/plgd-dev/device/schema/csr"
	"github.com/plgd-dev/device/schema/doxm"
	kitNet "github.com/plgd-dev/kit/v2/net"
	kitSecurity "github.com/plgd-dev/kit/v2/security"
)

//csr is encoded by PEM and returns PEM
type SignFunc = func(ctx context.Context, csr []byte) ([]byte, error)

type DialDTLS = func(ctx context.Context, addr string, dtlsCfg *dtls.Config, opts ...kitNetCoap.DialOptionFunc) (*kitNetCoap.ClientCloseHandler, error)
type DialTLS = func(ctx context.Context, addr string, tlsCfg *tls.Config, opts ...kitNetCoap.DialOptionFunc) (*kitNetCoap.ClientCloseHandler, error)

type Client struct {
	manufacturerCertificate tls.Certificate
	manufacturerCA          []*x509.Certificate
	dialDTLS                DialDTLS
	dialTLS                 DialTLS

	sign SignFunc
}

type OptionFunc func(Client) Client

func WithDialDTLS(dial DialDTLS) OptionFunc {
	return func(cfg Client) Client {
		if dial != nil {
			cfg.dialDTLS = dial
		}
		return cfg
	}
}

func WithDialTLS(dial DialTLS) OptionFunc {
	return func(cfg Client) Client {
		if dial != nil {
			cfg.dialTLS = dial
		}
		return cfg
	}
}

func NewClient(
	manufacturerCertificate tls.Certificate,
	manufacturerCA []*x509.Certificate,
	sign SignFunc,
	opts ...OptionFunc,
) *Client {
	c := Client{
		manufacturerCertificate: manufacturerCertificate,
		manufacturerCA:          manufacturerCA,
		sign:                    sign,
		dialDTLS:                kitNetCoap.DialUDPSecure,
		dialTLS:                 kitNetCoap.DialTCPSecure,
	}
	for _, o := range opts {
		c = o(c)
	}
	return &c
}

func (*Client) Type() doxm.OwnerTransferMethod {
	return doxm.ManufacturerCertificate
}

func (c *Client) Dial(ctx context.Context, addr kitNet.Addr, opts ...kitNetCoap.DialOptionFunc) (*kitNetCoap.ClientCloseHandler, error) {
	switch schema.Scheme(addr.GetScheme()) {
	case schema.UDPSecureScheme:
		rootCAs := x509.NewCertPool()
		for _, ca := range c.manufacturerCA {
			rootCAs.AddCert(ca)
		}

		tlsConfig := dtls.Config{
			InsecureSkipVerify:    true,
			CipherSuites:          []dtls.CipherSuiteID{dtls.TLS_ECDHE_ECDSA_WITH_AES_128_CCM_8, dtls.TLS_ECDHE_ECDSA_WITH_AES_128_CCM},
			Certificates:          []tls.Certificate{c.manufacturerCertificate},
			VerifyPeerCertificate: kitNetCoap.NewVerifyPeerCertificate(rootCAs, func(*x509.Certificate) error { return nil }),
		}
		return c.dialDTLS(ctx, addr.String(), &tlsConfig, opts...)
	case schema.TCPSecureScheme:
		rootCAs := x509.NewCertPool()
		for _, ca := range c.manufacturerCA {
			rootCAs.AddCert(ca)
		}
		tlsConfig := tls.Config{
			InsecureSkipVerify:    true,
			Certificates:          []tls.Certificate{c.manufacturerCertificate},
			VerifyPeerCertificate: kitNetCoap.NewVerifyPeerCertificate(rootCAs, func(*x509.Certificate) error { return nil }),
		}
		return c.dialTLS(ctx, addr.String(), &tlsConfig, opts...)
	}
	return nil, fmt.Errorf("cannot dial to url %v: scheme %v not supported", addr.URL(), addr.GetScheme())
}

func encodeToPem(encoding csr.CertificateEncoding, data []byte) []byte {
	if encoding == csr.CertificateEncoding_DER {
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: data})
	}
	return data
}

func (c *Client) ProvisionOwnerCredentials(ctx context.Context, tlsClient *kitNetCoap.ClientCloseHandler, ownerID, deviceID string) error {
	/*setup credentials - PostOwnerCredential*/
	var r csr.CertificateSigningRequestResponse
	err := tlsClient.GetResource(ctx, csr.ResourceURI, &r)
	if err != nil {
		return fmt.Errorf("cannot get csr for setup device owner credentials: %w", err)
	}

	pemCSR := encodeToPem(r.Encoding, r.CSR())

	signedCsr, err := c.sign(ctx, pemCSR)
	if err != nil {
		return fmt.Errorf("cannot sign csr for setup device owner credentials: %w", err)
	}

	certsFromChain, err := kitSecurity.ParseX509FromPEM(signedCsr)
	if err != nil {
		return fmt.Errorf("failed to parse chain of X509 certs: %w", err)
	}

	var deviceCredential credential.CredentialResponse
	err = tlsClient.GetResource(ctx, credential.ResourceURI, &deviceCredential, kitNetCoap.WithCredentialSubject(deviceID))
	if err != nil {
		return fmt.Errorf("cannot get device credential to setup device owner credentials: %w", err)
	}

	for _, cred := range deviceCredential.Credentials {
		switch {
		case cred.Usage == credential.CredentialUsage_CERT && cred.Type == credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
			cred.Usage == credential.CredentialUsage_TRUST_CA && cred.Type == credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE:
			err = tlsClient.DeleteResource(ctx, credential.ResourceURI, nil, kitNetCoap.WithCredentialId(cred.ID))
			if err != nil {
				return fmt.Errorf("cannot delete device credentials %v (%v) to setup device owner credentials: %w", cred.ID, cred.Usage, err)
			}
		}
	}

	setIdentityDeviceCredential := credential.CredentialUpdateRequest{
		ResourceOwner: ownerID,
		Credentials: []credential.Credential{
			{
				Subject: deviceID,
				Type:    credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
				Usage:   credential.CredentialUsage_CERT,
				PublicData: &credential.CredentialPublicData{
					DataInternal: string(signedCsr),
					Encoding:     credential.CredentialPublicDataEncoding_PEM,
				},
			},
			{
				Subject: ownerID,
				Type:    credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
				Usage:   credential.CredentialUsage_TRUST_CA,
				PublicData: &credential.CredentialPublicData{
					DataInternal: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certsFromChain[len(certsFromChain)-1].Raw})),
					Encoding:     credential.CredentialPublicDataEncoding_PEM,
				},
			},
		},
	}
	err = tlsClient.UpdateResource(ctx, credential.ResourceURI, setIdentityDeviceCredential, nil)
	if err != nil {
		return fmt.Errorf("cannot set device identity credentials: %w", err)
	}

	return nil
}
