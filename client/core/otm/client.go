package otm

import (
	"context"
	"encoding/pem"
	"fmt"

	kitNetCoap "github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema/credential"
	"github.com/plgd-dev/device/schema/csr"
	"github.com/plgd-dev/device/schema/doxm"
	kitNet "github.com/plgd-dev/kit/v2/net"
	kitSecurity "github.com/plgd-dev/kit/v2/security"
)

// csr is encoded by PEM and returns PEM
type SignFunc = func(ctx context.Context, csr []byte) ([]byte, error)

type Signer struct {
	Sign SignFunc
}

type Client interface {
	Type() doxm.OwnerTransferMethod
	Dial(ctx context.Context, addr kitNet.Addr, opts ...kitNetCoap.DialOptionFunc) (*kitNetCoap.ClientCloseHandler, error)
	ProvisionOwnerCredentials(ctx context.Context, client *kitNetCoap.ClientCloseHandler, ownerID, deviceID string) error
}

func encodeToPem(encoding csr.CertificateEncoding, data []byte) []byte {
	if encoding == csr.CertificateEncoding_DER {
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: data})
	}
	return data
}

func (c *Signer) ProvisionOwnerCredentials(ctx context.Context, tlsClient *kitNetCoap.ClientCloseHandler, ownerID, deviceID string) error {
	/*setup credentials - PostOwnerCredential*/
	var r csr.CertificateSigningRequestResponse
	err := tlsClient.GetResource(ctx, csr.ResourceURI, &r)
	if err != nil {
		return fmt.Errorf("cannot get csr for setup device owner credentials: %w", err)
	}

	pemCSR := encodeToPem(r.Encoding, r.CSR())

	signedCsr, err := c.Sign(ctx, pemCSR)
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
