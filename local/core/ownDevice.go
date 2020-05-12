package core

import (
	"context"
	"sync"

	"fmt"

	"github.com/go-ocf/go-coap/v2/udp/client"
	kitNet "github.com/go-ocf/kit/net"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/schema/acl"
	"github.com/go-ocf/sdk/schema/cloud"
)

type OTMClient interface {
	Type() schema.OwnerTransferMethod
	Dial(ctx context.Context, addr kitNet.Addr, opts ...kitNetCoap.DialOptionFunc) (*kitNetCoap.ClientCloseHandler, error)
	ProvisionOwnerCredentials(ctx context.Context, client *kitNetCoap.ClientCloseHandler, ownerID, deviceID string) error
}

/*
 * iotivityHack sets credential with pair-wise and removes it. It's needed to
 * enable ciphers for TLS communication with signed certificates.
 */
func iotivityHack(ctx context.Context, tlsClient *kitNetCoap.ClientCloseHandler, sdkID string) error {
	hackId := "52a201a7-824c-4fc6-9092-d2b6a3414a5b"

	setDeviceOwner := schema.DoxmUpdate{
		OwnerID: hackId,
	}

	/*doxm doesn't send any content for select OTM*/
	err := tlsClient.UpdateResource(ctx, "/oic/sec/doxm", setDeviceOwner, nil)
	if err != nil {
		return fmt.Errorf("cannot set device hackid as owner %w", err)
	}

	iotivityHackCredential := schema.CredentialUpdateRequest{
		ResourceOwner: sdkID,
		Credentials: []schema.Credential{
			schema.Credential{
				Subject: hackId,
				Type:    schema.CredentialType_SYMMETRIC_PAIR_WISE,
				PrivateData: &schema.CredentialPrivateData{
					DataInternal: "IOTIVITY HACK",
					Encoding:     schema.CredentialPrivateDataEncoding_RAW,
				},
			},
		},
	}
	err = tlsClient.UpdateResource(ctx, "/oic/sec/cred", iotivityHackCredential, nil)
	if err != nil {
		return fmt.Errorf("cannot set iotivity-hack credential: %w", err)
	}

	err = tlsClient.DeleteResource(ctx, "/oic/sec/cred", nil, kitNetCoap.WithCredentialSubject(hackId))
	if err != nil {
		return fmt.Errorf("cannot delete iotivity-hack credential: %w", err)
	}

	return nil
}

type ownCfg struct {
	iotivityHack bool
}

type OwnOption = func(ownCfg) ownCfg

// WithIotivityHack set this option when device with iotivity 2.0 will be onboarded.
func WithIotivityHack() OwnOption {
	return func(o ownCfg) ownCfg {
		o.iotivityHack = true
		return o
	}
}

type connUpdateResourcer interface {
	UpdateResource(context.Context, string, interface{}, interface{}, ...kitNetCoap.OptionFunc) error
}

func setOTM(ctx context.Context, conn connUpdateResourcer, selectOwnerTransferMethod schema.OwnerTransferMethod) error {
	selectOTM := schema.DoxmUpdate{
		SelectOwnerTransferMethod: selectOwnerTransferMethod,
	}
	/*doxm doesn't send any content for update*/
	return conn.UpdateResource(ctx, "/oic/sec/doxm", selectOTM, nil)
}

type selectOTMHandler struct {
	deviceID string
	cancel   context.CancelFunc

	conn *kitNetCoap.Client
	lock sync.Mutex
	err  error
}

func newSelectOTMHandler(deviceID string, cancel context.CancelFunc) *selectOTMHandler {
	return &selectOTMHandler{deviceID: deviceID, cancel: cancel}
}

func (h *selectOTMHandler) Handle(ctx context.Context, clientConn *client.ClientConn, links schema.ResourceLinks) {
	h.lock.Lock()
	defer h.lock.Unlock()

	link, err := GetResourceLink(links, "/oic/d")
	if err != nil {
		h.err = err
		clientConn.Close()
		return
	}
	deviceID := link.GetDeviceID()
	if deviceID == "" {
		clientConn.Close()
		h.err = fmt.Errorf("cannot determine deviceID")
		return
	}
	if h.conn != nil || deviceID != h.deviceID {
		clientConn.Close()
		return
	}
	h.conn = kitNetCoap.NewClient(clientConn.Client())
	h.cancel()
}

func (h *selectOTMHandler) Error(err error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.err == nil {
		h.err = err
	}
}

func (h *selectOTMHandler) Err() error {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.err
}

func (d *Device) selectOTMViaDiscovery(ctx context.Context, selectOwnerTransferMethod schema.OwnerTransferMethod) error {
	multicastConn := DialDiscoveryAddresses(ctx, d.cfg.discoveryConfiguration, d.cfg.errFunc)
	defer func() {
		for _, conn := range multicastConn {
			conn.Close()
		}
	}()

	ctxSelect, cancel := context.WithCancel(ctx)
	defer cancel()

	h := newSelectOTMHandler(d.DeviceID(), cancel)
	err := DiscoverDevices(ctxSelect, multicastConn, h)
	if h.conn != nil {
		defer h.conn.Close()
		return setOTM(ctx, h.conn, selectOwnerTransferMethod)
	}
	if err != nil {
		return err
	}
	err = h.Err()
	if err != nil {
		return err
	}

	return fmt.Errorf("device not found")
}

func (d *Device) selectOTM(ctx context.Context, selectOwnerTransferMethod schema.OwnerTransferMethod, links schema.ResourceLinks) error {
	var coapAddr kitNet.Addr
	var coapAddrFound bool
	var err error
	for _, link := range links {
		if coapAddr, err = link.GetUDPAddr(); err == nil {
			coapAddrFound = true
			break
		}
	}
	if coapAddrFound {
		coapConn, err := kitNetCoap.DialUDP(ctx, coapAddr.String())
		if err != nil {
			return fmt.Errorf("cannot connect to %v for select OTM: %w", coapAddr.URL(), err)
		}
		defer coapConn.Close()
		return setOTM(ctx, coapConn, selectOwnerTransferMethod)
	}
	return d.selectOTMViaDiscovery(ctx, selectOwnerTransferMethod)
}

func (d *Device) setProvisionResourceOwner(ctx context.Context, links schema.ResourceLinks, ownerID string) error {
	link, err := GetResourceLink(links, "/oic/sec/pstat")
	if err != nil {
		return err
	}
	setOwnerProvisionState := schema.ProvisionStatusUpdateRequest{
		ResourceOwner: ownerID,
	}

	/*pstat set owner of resource*/
	return d.UpdateResource(ctx, link, setOwnerProvisionState, nil)
}

func (d *Device) setOwnerACL(ctx context.Context, links schema.ResourceLinks, ownerID string) error {
	link, err := GetResourceLink(links, "/oic/sec/acl2")
	if err != nil {
		return err
	}

	cloudResources := make([]acl.Resource, 0, 1)
	for _, href := range links.GetResourceHrefs(cloud.ConfigurationResourceType) {
		cloudResources = append(cloudResources, acl.Resource{
			Href:       href,
			Interfaces: []string{"*"},
		})
	}

	/*acl2 set owner of resource*/
	setOwnerAcl := acl.UpdateRequest{
		ResourceOwner: ownerID,
		AccessControlList: []acl.AccessControl{
			acl.AccessControl{
				Permission: acl.AllPermissions,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: ownerID,
					},
				},
				Resources: acl.AllResources,
			},
			acl.AccessControl{
				Permission: acl.AllPermissions,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: ownerID,
					},
				},
				Resources: cloudResources,
			},
		},
	}

	return d.UpdateResource(ctx, link, setOwnerAcl, nil)
}

// Own set ownership of device
func (d *Device) Own(
	ctx context.Context,
	links schema.ResourceLinks,
	otmClient OTMClient,
	options ...OwnOption,
) error {
	var cfg ownCfg
	const errMsg = "cannot own device: %w"
	for _, opt := range options {
		cfg = opt(cfg)
	}

	ownership, err := d.GetOwnership(ctx)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	sdkID, err := d.GetSdkOwnerID()
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot set device owner %w", err))
	}

	if ownership.Owned {
		if ownership.OwnerID == sdkID {
			return nil
		}
		return fmt.Errorf(errMsg, fmt.Errorf("device is already owned by %v", ownership.OwnerID))
	}

	//ownership := d.ownership
	var supportOtm bool
	for _, s := range ownership.SupportedOwnerTransferMethods {
		if s == otmClient.Type() {
			supportOtm = true
			break
		}
	}
	if !supportOtm {
		return fmt.Errorf(errMsg, fmt.Errorf("ownership transfer method '%v' is unsupported, supported are: %v", otmClient.Type(), ownership.SupportedOwnerTransferMethods))
	}

	err = d.selectOTM(ctx, otmClient.Type(), links)
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot select otm: %w", err))
	}

	var tlsClient *kitNetCoap.ClientCloseHandler
	var errors []error
	for _, link := range links {
		if tlsAddr, err := link.GetUDPSecureAddr(); err == nil {
			tlsClient, err = otmClient.Dial(ctx, tlsAddr, d.cfg.dialOptions...)
			if err == nil {
				break
			}
			errors = append(errors, fmt.Errorf("cannot connect to %v: %w", tlsAddr.URL(), err))
		}
		if tlsAddr, err := link.GetTCPSecureAddr(); err == nil {
			tlsClient, err = otmClient.Dial(ctx, tlsAddr, d.cfg.dialOptions...)
			if err == nil {
				break
			}
			errors = append(errors, fmt.Errorf("cannot connect to %v: %w", tlsAddr.URL(), err))
		}
	}
	if tlsClient == nil {
		if len(errors) == 0 {
			return fmt.Errorf(errMsg, fmt.Errorf("cannot get udp/tcp secure address: not found"))
		}
		return fmt.Errorf(errMsg, fmt.Errorf("cannot get udp/tcp secure address: %+v", errors))
	}

	var provisionState schema.ProvisionStatusResponse
	err = tlsClient.GetResource(ctx, "/oic/sec/pstat", &provisionState)
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot get provision state %w", err))
	}

	if provisionState.DeviceOnboardingState.Pending {
		return fmt.Errorf(errMsg, fmt.Errorf("device pending for operation state %v", provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState))
	}

	if provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState != schema.OperationalState_RFOTM {
		return fmt.Errorf(errMsg, fmt.Errorf("device operation state %v is not %v", provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState, schema.OperationalState_RFOTM))
	}

	if !provisionState.SupportedOperationalModes.Has(schema.OperationalMode_CLIENT_DIRECTED) {
		return fmt.Errorf(errMsg, fmt.Errorf("device supports %v, but only %v is supported", provisionState.SupportedOperationalModes, schema.OperationalMode_CLIENT_DIRECTED))
	}

	updateProvisionState := schema.ProvisionStatusUpdateRequest{
		CurrentOperationalMode: schema.OperationalMode_CLIENT_DIRECTED,
	}
	/*pstat doesn't send any content for select OperationalMode*/
	err = tlsClient.UpdateResource(ctx, "/oic/sec/pstat", updateProvisionState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot update provision state %w", err))
	}

	/*setup credentials */
	err = otmClient.ProvisionOwnerCredentials(ctx, tlsClient, sdkID, d.DeviceID())
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot provision owner %w", err))
	}

	/*
	 * THIS IS HACK FOR iotivity -> enables ciphers for TLS communication with signed certificates.
	 * Tested with iotivity 2.0.1-RC0.
	 */
	if cfg.iotivityHack {
		err = iotivityHack(ctx, tlsClient, sdkID)
		if err != nil {
			return fmt.Errorf(errMsg, err)
		}
	}
	// END OF HACK

	setDeviceOwner := schema.DoxmUpdate{
		OwnerID: sdkID,
	}

	/*doxm doesn't send any content for select OTM*/
	err = tlsClient.UpdateResource(ctx, "/oic/sec/doxm", setDeviceOwner, nil)
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot set device owner %w", err))
	}

	/*verify ownership*/
	var verifyOwner schema.Doxm
	err = tlsClient.GetResource(ctx, "/oic/sec/doxm", &verifyOwner)
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot verify owner: %w", err))
	}
	if verifyOwner.OwnerID != sdkID {
		return fmt.Errorf(errMsg, err)
	}

	setDeviceOwned := schema.DoxmUpdate{
		ResourceOwner: sdkID,
		DeviceID:      d.DeviceID(),
		Owned:         true,
	}

	/*doxm doesn't send any content for select OTM*/
	err = tlsClient.UpdateResource(ctx, "/oic/sec/doxm", setDeviceOwned, nil)
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot set device owned %w", err))
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

	/*pstat set owner of resource*/
	err = d.setProvisionResourceOwner(ctx, links, sdkID)
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot update provision state resource owner to setup device owner ACLs: %w", err))
	}

	/*acl2 set owner of resource*/
	err = d.setOwnerACL(ctx, links, sdkID)
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot update acl resource owner: %w", err))
	}

	// Provision the device to switch back to normal operation.
	p, err := d.Provision(ctx, links)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	err = p.Close(ctx)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	d.Close(ctx)

	return nil
}
