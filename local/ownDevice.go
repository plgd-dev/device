package local

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"sync"
	"encoding/base64"
	"time"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/cbor"
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

type OTMClient interface {
	Type() schema.OwnerTransferMethod
	Dial(ctx context.Context, addr string) (*CoapClient, error)
	// SignCSR
	SignCSR(encoding schema.CertificateEncoding, csr []byte) (signedEncoding schema.CertificateEncoding, signedCsr []byte, err error)
}

type ManufacturerOTMClient struct {
	manufacturerCertificate   tls.Certificate
	manufacturerCA            *x509.Certificate
	manufacturerCAKey         *ecdsa.PrivateKey
	signedCertificateValidFor time.Duration
}

func NewManufacturerOTMClient(manufacturerCertificate tls.Certificate, manufacturerCA *x509.Certificate, manufacturerCAKey *ecdsa.PrivateKey, signedCertificateValidFor time.Duration) *ManufacturerOTMClient {
	return &ManufacturerOTMClient{
		manufacturerCertificate:   manufacturerCertificate,
		manufacturerCA:            manufacturerCA,
		manufacturerCAKey:         manufacturerCAKey,
		signedCertificateValidFor: signedCertificateValidFor,
	}
}

func (*ManufacturerOTMClient) Type() schema.OwnerTransferMethod {
	return schema.ManufacturerCertificate
}

func (otmc *ManufacturerOTMClient) Dial(ctx context.Context, addr string) (*CoapClient, error) {
	return DialTcpTls(ctx, addr, otmc.manufacturerCertificate, []*x509.Certificate{otmc.manufacturerCA}, func(*x509.Certificate)error {return nil})
}

func (otmc *ManufacturerOTMClient) SignCSR(encoding schema.CertificateEncoding, csr []byte) (signedEncoding schema.CertificateEncoding, signedCsr []byte, err error) {
	der := csr
	if encoding == schema.CertificateEncoding_PEM {
		derBlock, _ := pem.Decode(csr)
		if derBlock == nil {
			err = fmt.Errorf("invalid encoding for csr")
			return
		}
		der = derBlock.Bytes
	}

	certificateRequest, err := x509.ParseCertificateRequest(der)
	if err != nil {
		return
	}

	err = certificateRequest.CheckSignature()
	if err != nil {
		return
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(otmc.signedCertificateValidFor)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	template := x509.Certificate{
		SerialNumber:       serialNumber,
		NotBefore:          notBefore,
		NotAfter:           notAfter,
		Subject:            certificateRequest.Subject,
		PublicKeyAlgorithm: certificateRequest.PublicKeyAlgorithm,
		PublicKey:          certificateRequest.PublicKey,
		SignatureAlgorithm: certificateRequest.SignatureAlgorithm,
		DNSNames:           certificateRequest.DNSNames,
		IPAddresses:        certificateRequest.IPAddresses,
		Extensions:         certificateRequest.Extensions,
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		UnknownExtKeyUsage: []asn1.ObjectIdentifier{ekuOcfId},
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	signedEncoding = schema.CertificateEncoding_DER
	signedCsr, err = x509.CreateCertificate(rand.Reader, &template, otmc.manufacturerCA, certificateRequest.PublicKey, otmc.manufacturerCAKey)
	return
}

func encodeToBase64(encoding schema.CertificateEncoding, data []byte) string {
	switch encoding {
	case schema.CertificateEncoding_DER:
		return base64.StdEncoding.EncodeToString(data)
	}
	return string(data)
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
/*
	if ownership.Owned {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("device already owned by %v", ownership.DeviceOwner))
	}
*/
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

	selectOTM := schema.DoxmUpdate{
		SelectOwnerTransferMethod: otmClient.Type(),
	}

	/*doxm doesn't send any content for select OTM*/
	err = client.UpdateResourceCBOR(ctx, "/oic/sec/doxm", selectOTM, nil)
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

	tlsClient, err := otmClient.Dial(ctx, tcpTLSAddr.String())
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot create TLS connection: %v", err))
	}

	var provisionState schema.ProvisionStatusResponse
	err = tlsClient.GetResourceCBOR(ctx, "/oic/sec/pstat", &provisionState)
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
	err = tlsClient.UpdateResourceCBOR(ctx, "/oic/sec/pstat", updateProvisionState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot update provision state %v", err))
	}

	sdkId, err := c.GetSdkId()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot set device owner %v", err))
	}

	setDeviceOwner := schema.DoxmUpdate{
		DeviceOwner: sdkId,
		ResourceOwner: sdkId,
		Owned: true,
	}

	/*doxm doesn't send any content for select OTM*/
	err = tlsClient.UpdateResourceCBOR(ctx, "/oic/sec/doxm", setDeviceOwner, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot set device owner %v", err))
	}

	/*verify ownership*/
	var verifyOwner schema.Doxm
	err = tlsClient.GetResourceCBOR(ctx, "/oic/sec/doxm", &verifyOwner)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}
	if verifyOwner.DeviceOwner != sdkId {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("device is owned by %v, not by %v", verifyOwner.DeviceOwner, sdkId))
	}

	/*setup credentials - PostOwnerCredential*/
	var csr schema.CertificateSigningRequestResponse
	err = tlsClient.GetResourceCBOR(ctx, "/oic/sec/csr", &csr)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get csr for setup device owner credentials: %v", err))
	}
	signedEncodingCsr, signedCsr, err := otmClient.SignCSR(csr.Encoding, csr.CertificateSigningRequest)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot sign csr for setup device owner credentials: %v", err))
	}

	var deviceCredential schema.CredentialResponse
	err = tlsClient.GetResourceCBOR(ctx, "/oic/sec/cred", &deviceCredential, WithCredentialSubject(deviceID))
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get device credential to setup device owner credentials: %v", err))
	}

	for _, cred := range deviceCredential.Credentials {
		switch {
		case cred.Usage == schema.CredentialUsage_CERT && cred.Type == schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
		cred.Usage == schema.CredentialUsage_TRUST_CA && cred.Type == schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE: 
			err = tlsClient.DeleteResourceCBOR(ctx, "/oic/sec/cred", nil, WithCredentialId(cred.ID))
			if err != nil {
				return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot delete device credentials %v (%v) to setup device owner credentials: %v", cred.ID, cred.Usage, err))
			}
		}
	}

	setDeviceCredential := schema.CredentialUpdateRequest{
		ResourceOwner: sdkId,
		Credentials: []schema.Credential{
			schema.Credential{
				Subject: deviceID,
				Type:    schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
				Usage:   schema.CredentialUsage_CERT,
				PublicData: schema.CredentialPublicData{
					Data:     encodeToBase64(signedEncodingCsr, signedCsr),
					Encoding: schema.CredentialPublicDataEncoding(signedEncodingCsr),
				},
			},
		},
	}
	err = tlsClient.UpdateResourceCBOR(ctx, "/oic/sec/cred", setDeviceCredential, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot set device credentials: %v", err))
	}


	cas, err := c.GetCertificateAuthorities()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get CAs to setup device owner credentials: %v", err))
	}

	for _, ca := range cas {
		setCaCredential := schema.CredentialUpdateRequest{
			ResourceOwner: sdkId,
			Credentials: []schema.Credential{
				schema.Credential{
					Subject: sdkId,
					Type:    schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
					Usage:   schema.CredentialUsage_TRUST_CA,
					PublicData: schema.CredentialPublicData{
						Data:     encodeToBase64(schema.CertificateEncoding_DER, ca.Raw),
						Encoding: schema.CredentialPublicDataEncoding_DER,
					},
				},
			},
		}
		err = tlsClient.UpdateResourceCBOR(ctx, "/oic/sec/cred", setCaCredential, nil)
		if err != nil {
			return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot set device CA credentials: %v", err))
		}
	}


	//For Servers based on OCF 1.0, PostOwnerAcl can be executed using
    //the already-existing session. However, get ready here to use the
    //Owner Credential for establishing future secure sessions.
    //
    //For Servers based on OIC 1.1, PostOwnerAcl might fail with status
    //OC_STACK_UNAUTHORIZED_REQ. After such a failure, OwnerAclHandler
    //will close the current session and re-establish a new session,
    //using the Owner Credential.



	setOwnerProvisionState := schema.ProvisionStatusUpdateRequest{
		ResourceOwner: sdkId,
	}
	/*pstat set owner of resource*/
	err = tlsClient.UpdateResourceCBOR(ctx, "/oic/sec/pstat", setOwnerProvisionState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot update provision state resource owner to setup device owner ACLs: %v", err))
	}

	/*acl2 set owner of resource*/
	setOwnerAcl := schema.AccessControlListUpdateRequest{
		ResourceOwner: sdkId,
		AccessControlList: []schema.AccessControl{
			schema.AccessControl{
				Permission: schema.AccessControlPermission_CREATE | schema.AccessControlPermission_READ | schema.AccessControlPermission_WRITE | schema.AccessControlPermission_DELETE | schema.AccessControlPermission_NOTIFY,
				Subject: schema.AccessControlSubject{
					AccessControlSubjectDevice: &schema.AccessControlSubjectDevice{
						DeviceId: sdkId,
					},
				},
				Resources: []schema.AccessControlResource{
					schema.AccessControlResource{
						Wildcard: schema.AccessControlResourceWildcard_NONCFG_ALL,
					},
				},
			},
		},
	}


	v, _ := cbor.ToJSON(func () []byte {
		v, _ := cbor.Encode(setOwnerAcl)
		return v
	}())
	fmt.Println(v)

	err = tlsClient.UpdateResourceCBOR(ctx, "/oic/sec/acl2", setOwnerAcl, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot update acl resource owner: %v", err))
	}

	// Change the dos.s value to RFPRO
	setProvisionStateToRFPRO := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RFPRO,
		},
	}

	err = tlsClient.UpdateResourceCBOR(ctx, "/oic/sec/pstat", setProvisionStateToRFPRO, nil)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot update provision state to RFPRO to setup device owner ACLs: %v", err))
	}

	cert,err := c.GetCertificate()
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot get cert to setup device owner ACLs: %v", err))
	}

	_, err = DialTcpTls(ctx, tcpTLSAddr.String(), cert, cas, VerifyIndetityCertificate)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot create TLS communicaton to setup device owner ACLs: %v", err))
	}

	return nil
}
