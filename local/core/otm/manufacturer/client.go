package manufacturer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/pion/dtls/v2"
	kitNet "github.com/plgd-dev/kit/net"
	kitNetCoap "github.com/plgd-dev/kit/net/coap"
	kitSecurity "github.com/plgd-dev/kit/security"
	"github.com/plgd-dev/sdk/schema"
)

type CertificateSigner = interface {
	//csr is encoded by PEM and returns PEM
	Sign(ctx context.Context, csr []byte) ([]byte, error)
}

type Client struct {
	manufacturerCertificate tls.Certificate
	manufacturerCA          []*x509.Certificate
	disableDTLS             bool
	disableTCPTLS           bool

	signer CertificateSigner
}

type OptionFunc func(Client) Client

func WithoutTCPTLS() OptionFunc {
	return func(cfg Client) Client {
		cfg.disableTCPTLS = true
		return cfg
	}
}

func WithoutDTLS() OptionFunc {
	return func(cfg Client) Client {
		cfg.disableDTLS = true
		return cfg
	}
}

func NewClient(
	manufacturerCertificate tls.Certificate,
	manufacturerCA []*x509.Certificate,
	signer CertificateSigner,
	opts ...OptionFunc,
) *Client {
	c := Client{
		manufacturerCertificate: manufacturerCertificate,
		manufacturerCA:          manufacturerCA,
		signer:                  signer,
	}
	for _, o := range opts {
		c = o(c)
	}
	return &c
}

func (*Client) Type() schema.OwnerTransferMethod {
	return schema.ManufacturerCertificate
}

func (c *Client) Dial(ctx context.Context, addr kitNet.Addr, opts ...kitNetCoap.DialOptionFunc) (*kitNetCoap.ClientCloseHandler, error) {
	switch schema.Scheme(addr.GetScheme()) {
	case schema.UDPSecureScheme:
		if c.disableDTLS {
			return nil, fmt.Errorf("dtls is disabled")
		}
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
		return kitNetCoap.DialUDPSecure(ctx, addr.String(), &tlsConfig, opts...)
	case schema.TCPSecureScheme:
		if c.disableTCPTLS {
			return nil, fmt.Errorf("tcp-tls is disabled")
		}
		rootCAs := x509.NewCertPool()
		for _, ca := range c.manufacturerCA {
			rootCAs.AddCert(ca)
		}
		tlsConfig := tls.Config{
			InsecureSkipVerify:    true,
			Certificates:          []tls.Certificate{c.manufacturerCertificate},
			VerifyPeerCertificate: kitNetCoap.NewVerifyPeerCertificate(rootCAs, func(*x509.Certificate) error { return nil }),
		}
		return kitNetCoap.DialTCPSecure(ctx, addr.String(), &tlsConfig, opts...)
	}
	return nil, fmt.Errorf("cannot dial to url %v: scheme %v not supported", addr.URL(), addr.GetScheme())
}

func encodeToPem(encoding schema.CertificateEncoding, data []byte) []byte {
	if encoding == schema.CertificateEncoding_DER {
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: data})
	}
	return data
}

func (c *Client) ProvisionOwnerCredentials(ctx context.Context, tlsClient *kitNetCoap.ClientCloseHandler, ownerID, deviceID string) error {
	/*setup credentials - PostOwnerCredential*/
	var csr schema.CertificateSigningRequestResponse
	err := tlsClient.GetResource(ctx, "/oic/sec/csr", &csr)
	if err != nil {
		return fmt.Errorf("cannot get csr for setup device owner credentials: %w", err)
	}

	pemCSR := encodeToPem(csr.Encoding, csr.CSR())

	signedCsr, err := c.signer.Sign(ctx, pemCSR)
	if err != nil {
		return fmt.Errorf("cannot sign csr for setup device owner credentials: %w", err)
	}

	certsFromChain, err := kitSecurity.ParseX509FromPEM(signedCsr)
	if err != nil {
		return fmt.Errorf("Failed to parse chain of X509 certs: %w", err)
	}

	var deviceCredential schema.CredentialResponse
	err = tlsClient.GetResource(ctx, "/oic/sec/cred", &deviceCredential, kitNetCoap.WithCredentialSubject(deviceID))
	if err != nil {
		return fmt.Errorf("cannot get device credential to setup device owner credentials: %w", err)
	}

	for _, cred := range deviceCredential.Credentials {
		switch {
		case cred.Usage == schema.CredentialUsage_CERT && cred.Type == schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
			cred.Usage == schema.CredentialUsage_TRUST_CA && cred.Type == schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE:
			err = tlsClient.DeleteResource(ctx, "/oic/sec/cred", nil, kitNetCoap.WithCredentialId(cred.ID))
			if err != nil {
				return fmt.Errorf("cannot delete device credentials %v (%v) to setup device owner credentials: %w", cred.ID, cred.Usage, err)
			}
		}
	}

	setIdentityDeviceCredential := schema.CredentialUpdateRequest{
		ResourceOwner: ownerID,
		Credentials: []schema.Credential{
			schema.Credential{
				Subject: deviceID,
				Type:    schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
				Usage:   schema.CredentialUsage_CERT,
				PublicData: &schema.CredentialPublicData{
					DataInternal: string(signedCsr),
					Encoding:     schema.CredentialPublicDataEncoding_PEM,
				},
			},
			schema.Credential{
				Subject: ownerID,
				Type:    schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
				Usage:   schema.CredentialUsage_TRUST_CA,
				PublicData: &schema.CredentialPublicData{
					DataInternal: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certsFromChain[len(certsFromChain)-1].Raw})),
					Encoding:     schema.CredentialPublicDataEncoding_PEM,
				},
			},
		},
	}
	err = tlsClient.UpdateResource(ctx, "/oic/sec/cred", setIdentityDeviceCredential, nil)
	if err != nil {
		return fmt.Errorf("cannot set device identity credentials: %w", err)
	}

	return nil
}

/*
func (c *Client) SignCertificate(ctx context.Context, csr []byte) (signedCsr []byte, err error) {
	return c.signer.Sign(ctx, csr)
}
*/
