package local

import (
	"context"

	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"sync"
	//"encoding/base64"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"
)

type deviceOwnershipClient struct {
	*coapClient
	ownership schema.Doxm
}

func (c *deviceOwnershipClient) GetOwnership() schema.Doxm {
	return c.ownership
}

func (c *deviceOwnershipClient) GetTcpSecureAddress(ctx context.Context, deviceID string) (string, error) {
	deviceLink, err := c.GetDeviceLinks(ctx, deviceID)
	if err != nil {
		return "", err
	}
	var resourceLink schema.ResourceLink
	for _, link := range deviceLink.Links {
		if link.HasType("oic.wk.d") {
			resourceLink = link
			break
		}
	}
	tcpTLSAddr, err := resourceLink.GetTCPSecureAddr()
	if err != nil {
		return "", err
	}
	return tcpTLSAddr.String(), nil
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
	if ownership.DeviceId == h.deviceID && h.client == nil {
		h.client = &deviceOwnershipClient{coapClient: NewCoapClient(clientConn, schema.UDPScheme), ownership: ownership}
		h.cancel()
	}
}

func (h *deviceOwnershipHandler) Error(err error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.err == nil {
		h.err = err
	}
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

func (c *Client) ownDeviceFindClient(ctx context.Context, deviceID string, status resource.DiscoverOwnershipStatus) (*deviceOwnershipClient, error) {
	ctxOwn, cancel := context.WithCancel(ctx)
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
	err = h.Err()
	if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("device not found")
}

type OTMClient interface {
	Type() schema.OwnerTransferMethod
	Dial(ctx context.Context, addr string) (*coapClient, error)
	// SignCSR
	SignCertificate(ctx context.Context, csr []byte) (signedCsr []byte, err error)
}

type CertificateSigner interface {
	//csr is encoded by DER
	Sign(ctx context.Context, csr []byte) ([]byte, error)
}

type ManufacturerOTMClient struct {
	manufacturerCertificate tls.Certificate
	manufacturerCA          *x509.Certificate

	signer CertificateSigner
}

func NewManufacturerOTMClient(manufacturerCertificate tls.Certificate, manufacturerCA *x509.Certificate, signer CertificateSigner) *ManufacturerOTMClient {
	return &ManufacturerOTMClient{
		manufacturerCertificate: manufacturerCertificate,
		manufacturerCA:          manufacturerCA,
		signer:                  signer,
	}
}

func (*ManufacturerOTMClient) Type() schema.OwnerTransferMethod {
	return schema.ManufacturerCertificate
}

func (otmc *ManufacturerOTMClient) Dial(ctx context.Context, addr string) (*coapClient, error) {
	return DialTcpTls(ctx, addr, otmc.manufacturerCertificate, []*x509.Certificate{otmc.manufacturerCA}, func(*x509.Certificate) error { return nil })
}

func encodeToDer(encoding schema.CertificateEncoding, data []byte) ([]byte, error) {
	der := data
	if encoding == schema.CertificateEncoding_PEM {
		derBlock, _ := pem.Decode(data)
		if derBlock == nil {
			return nil, fmt.Errorf("invalid pem encoding")
		}
		der = derBlock.Bytes
	}
	return der, nil
}

func (otmc *ManufacturerOTMClient) SignCertificate(ctx context.Context, csr []byte) (signedCsr []byte, err error) {
	return otmc.signer.Sign(ctx, csr)
}

func encodeToPem(encoding schema.CertificateEncoding, data []byte) string {
	switch encoding {
	case schema.CertificateEncoding_DER:
		d := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
		d = append(d, []byte{0}...)
		return string(d)
	}
	return string(data)
}

/*
 * iotivityHack sets credential with pair-wise and removes it. It's needed to
 * enable ciphers for TLS communication with signed certificates.
 */
func iotivityHack(ctx context.Context, tlsClient *coapClient, sdkID string) error {
	hackId := "52a201a7-824c-4fc6-9092-d2b6a3414a5b"

	setDeviceOwner := schema.DoxmUpdate{
		DeviceOwner: hackId,
	}

	/*doxm doesn't send any content for select OTM*/
	err := tlsClient.UpdateResource(ctx, "/oic/sec/doxm", setDeviceOwner, nil)
	if err != nil {
		return fmt.Errorf("cannot set device hackid as owner %v", err)
	}

	iotivityHackCredential := schema.CredentialUpdateRequest{
		ResourceOwner: sdkID,
		Credentials: []schema.Credential{
			schema.Credential{
				Subject: hackId,
				Type:    schema.CredentialType_SYMMETRIC_PAIR_WISE,
				PrivateData: schema.CredentialPrivateData{
					Data:     "IOTIVITY HACK",
					Encoding: schema.CredentialPrivateDataEncoding_RAW,
				},
			},
		},
	}
	err = tlsClient.UpdateResource(ctx, "/oic/sec/cred", iotivityHackCredential, nil)
	if err != nil {
		return fmt.Errorf("cannot set iotivity-hack credential: %v", err)
	}

	err = tlsClient.DeleteResource(ctx, "/oic/sec/cred", nil, WithCredentialSubject(hackId))
	if err != nil {
		return fmt.Errorf("cannot delete iotivity-hack credential: %v", err)
	}

	return nil
}

// OwnDevice set ownership of device
func (c *Client) OwnDevice(
	ctx context.Context,
	deviceID string,
	otmClient OTMClient,
) error {
	const errMsg = "cannot own device %v: %v"

	client, err := c.ownDeviceFindClient(ctx, deviceID, resource.DiscoverAllDevices)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	ownership := client.GetOwnership()
	var supportOtm bool
	for _, s := range ownership.SupportedOwnerTransferMethods {
		if s == otmClient.Type() {
			supportOtm = true
			break
		}
	}
	if !supportOtm {
		return fmt.Errorf(errMsg, deviceID, fmt.Sprintf("ownership transfer method '%v' is unsupported, supported are: %v", otmClient.Type(), ownership.SupportedOwnerTransferMethods))
	}

	selectOTM := schema.DoxmUpdate{
		SelectOwnerTransferMethod: otmClient.Type(),
	}

	/*doxm doesn't send any content for select OTM*/
	fmt.Printf("SELECT OTM START\n")
	err = client.UpdateResource(ctx, "/oic/sec/doxm", selectOTM, nil)
	fmt.Printf("SELECT OTM STOP\n")
	if err != nil {
		if ownership.Owned {
			return fmt.Errorf(errMsg, deviceID, fmt.Errorf("device is already owned by %v", ownership.DeviceOwner))
		}
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot select OTM: %v", err))
	}

	deviceClient, err := c.GetDevice(ctx, deviceID, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}
	links := deviceClient.GetResourceLinks()
	if len(links) == 0 {
		return fmt.Errorf(errMsg, deviceID, "device links are empty")
	}
	tlsAddr, err := deviceClient.GetResourceLinks()[0].GetTCPSecureAddr()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, "device links are empty")
	}

	tlsClient, err := otmClient.Dial(ctx, tlsAddr.String())
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot create TLS connection: %v", err))
	}

	var provisionState schema.ProvisionStatusResponse
	err = tlsClient.GetResource(ctx, "/oic/sec/pstat", &provisionState)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get provision state %v", err))
	}

	if provisionState.DeviceOnboardingState.Pending {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("device pending for operation state %v", provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState))
	}

	if provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState != schema.OperationalState_RFOTM {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("device operation state %v is not %v", provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState, schema.OperationalState_RFOTM))
	}

	if !provisionState.SupportedOperationalModes.Has(schema.OperationalMode_CLIENT_DIRECTED) {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("device supports %v, but only %v is supported", provisionState.SupportedOperationalModes, schema.OperationalMode_CLIENT_DIRECTED))
	}

	updateProvisionState := schema.ProvisionStatusUpdateRequest{
		CurrentOperationalMode: schema.OperationalMode_CLIENT_DIRECTED,
	}
	/*pstat doesn't send any content for select OperationalMode*/
	err = tlsClient.UpdateResource(ctx, "/oic/sec/pstat", updateProvisionState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot update provision state %v", err))
	}

	sdkID, err := c.GetSdkDeviceID()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot set device owner %v", err))
	}

	/*setup credentials - PostOwnerCredential*/
	var csr schema.CertificateSigningRequestResponse
	err = tlsClient.GetResource(ctx, "/oic/sec/csr", &csr)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get csr for setup device owner credentials: %v", err))
	}

	der, err := encodeToDer(csr.Encoding, csr.CertificateSigningRequest)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot encode csr to der: %v", err))
	}

	signedCsr, err := otmClient.SignCertificate(ctx, der)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot sign csr for setup device owner credentials: %v", err))
	}

	var deviceCredential schema.CredentialResponse
	err = tlsClient.GetResource(ctx, "/oic/sec/cred", &deviceCredential, WithCredentialSubject(deviceID))
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get device credential to setup device owner credentials: %v", err))
	}

	for _, cred := range deviceCredential.Credentials {
		switch {
		case cred.Usage == schema.CredentialUsage_CERT && cred.Type == schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
			cred.Usage == schema.CredentialUsage_TRUST_CA && cred.Type == schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE:
			err = tlsClient.DeleteResource(ctx, "/oic/sec/cred", nil, WithCredentialId(cred.ID))
			if err != nil {
				return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot delete device credentials %v (%v) to setup device owner credentials: %v", cred.ID, cred.Usage, err))
			}
		}
	}

	setIdentityDeviceCredential := schema.CredentialUpdateRequest{
		ResourceOwner: sdkID,
		Credentials: []schema.Credential{
			schema.Credential{
				Subject: deviceID,
				Type:    schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
				Usage:   schema.CredentialUsage_CERT,
				PublicData: schema.CredentialPublicData{
					Data:     string(signedCsr),
					Encoding: schema.CredentialPublicDataEncoding_DER,
				},
			},
		},
	}
	err = tlsClient.UpdateResource(ctx, "/oic/sec/cred", setIdentityDeviceCredential, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot set device identity credentials: %v", err))
	}

	cas, err := c.GetCertificateAuthorities()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get CAs to setup device owner credentials: %v", err))
	}

	for _, ca := range cas {
		setCaCredential := schema.CredentialUpdateRequest{
			ResourceOwner: sdkID,
			Credentials: []schema.Credential{
				schema.Credential{
					Subject: sdkID,
					Type:    schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
					Usage:   schema.CredentialUsage_TRUST_CA,
					PublicData: schema.CredentialPublicData{
						Data:     string(ca.Raw),
						Encoding: schema.CredentialPublicDataEncoding_DER,
					},
				},
			},
		}
		err = tlsClient.UpdateResource(ctx, "/oic/sec/cred", setCaCredential, nil)
		if err != nil {
			return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot set device CA credentials: %v", err))
		}
	}

	/*
	 * THIS IS HACK FOR iotivity -> enables ciphers for TLS communication with signed certificates.
	 * Tested with iotivity 2.0.1-RC0.
	 */
	isIotivity, err := tlsClient.IsIotivity(ctx)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}
	if isIotivity {
		err = iotivityHack(ctx, tlsClient, sdkID)
		if err != nil {
			return fmt.Errorf(errMsg, deviceID, err)
		}
	}
	// END OF HACK

	setDeviceOwner := schema.DoxmUpdate{
		DeviceOwner: sdkID,
	}

	/*doxm doesn't send any content for select OTM*/
	err = tlsClient.UpdateResource(ctx, "/oic/sec/doxm", setDeviceOwner, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot set device owner %v", err))
	}

	/*verify ownership*/
	var verifyOwner schema.Doxm
	err = tlsClient.GetResource(ctx, "/oic/sec/doxm", &verifyOwner)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot verify owner: %v", err))
	}
	if verifyOwner.DeviceOwner != sdkID {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	setDeviceOwned := schema.DoxmUpdate{
		ResourceOwner: sdkID,
		DeviceId:      deviceID,
		Owned:         true,
	}

	/*doxm doesn't send any content for select OTM*/
	err = tlsClient.UpdateResource(ctx, "/oic/sec/doxm", setDeviceOwned, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot set device owned %v", err))
	}

	//For Servers based on OCF 1.0, PostOwnerAcl can be executed using
	//the already-existing session. However, get ready here to use the
	//Owner Credential for establishing future secure sessions.
	//
	//For Servers based on OIC 1.1, PostOwnerAcl might fail with status
	//OC_STACK_UNAUTHORIZED_REQ. After such a failure, OwnerAclHandler
	//will close the current session and re-establish a new session,
	//using the Owner Credential.

	tlsClient.Close()

	setOwnerProvisionState := schema.ProvisionStatusUpdateRequest{
		ResourceOwner: sdkID,
	}

	/*pstat set owner of resource*/
	err = c.UpdateResource(ctx, deviceID, "/oic/sec/pstat", setOwnerProvisionState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot update provision state resource owner to setup device owner ACLs: %v", err))
	}

	/*acl2 set owner of resource*/
	setOwnerAcl := schema.AccessControlListUpdateRequest{
		ResourceOwner: sdkID,
		AccessControlList: []schema.AccessControl{
			schema.AccessControl{
				Permission: schema.AccessControlPermission_CREATE | schema.AccessControlPermission_READ | schema.AccessControlPermission_WRITE | schema.AccessControlPermission_DELETE | schema.AccessControlPermission_NOTIFY,
				Subject: schema.AccessControlSubject{
					AccessControlSubjectDevice: &schema.AccessControlSubjectDevice{
						DeviceId: sdkID,
					},
				},
				Resources: []schema.AccessControlResource{
					schema.AccessControlResource{
						Interfaces: []string{"*"},
						Wildcard:   schema.AccessControlResourceWildcard_NONCFG_ALL,
					},
				},
			},
		},
	}

	err = c.UpdateResource(ctx, deviceID, "/oic/sec/acl2", setOwnerAcl, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot update acl resource owner: %v", err))
	}

	// Change the dos.s value to RFPRO
	setProvisionStateToRFPRO := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RFPRO,
		},
	}

	err = c.UpdateResource(ctx, deviceID, "/oic/sec/pstat", setProvisionStateToRFPRO, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot update provision state to RFPRO to setup device owner ACLs: %v", err))
	}

	// Change the dos.s value to RFNOP
	setProvisionStateToRFNOP := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RFNOP,
		},
	}

	err = c.UpdateResource(ctx, deviceID, "/oic/sec/pstat", setProvisionStateToRFNOP, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot update provision state to RFNOP to setup device owner ACLs: %v", err))
	}

	return nil
}
