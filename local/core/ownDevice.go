package core

import (
	"context"
	"sync"
	"time"

	"fmt"

	"github.com/plgd-dev/go-coap/v2/udp/client"
	kitNet "github.com/plgd-dev/kit/net"
	kitNetCoap "github.com/plgd-dev/kit/net/coap"
	"github.com/plgd-dev/sdk/schema"
	"github.com/plgd-dev/sdk/schema/acl"
	"github.com/plgd-dev/sdk/schema/cloud"
)

type OTMClient interface {
	Type() schema.OwnerTransferMethod
	Dial(ctx context.Context, addr kitNet.Addr, opts ...kitNetCoap.DialOptionFunc) (*kitNetCoap.ClientCloseHandler, error)
	ProvisionOwnerCredentials(ctx context.Context, client *kitNetCoap.ClientCloseHandler, ownerID, deviceID string) error
}

type ActionDuringOwnFunc = func(ctx context.Context, client *kitNetCoap.ClientCloseHandler) (string, error)

type ownCfg struct {
	actionDuringOwn ActionDuringOwnFunc
}

type OwnOption = func(ownCfg) ownCfg

// WithActionDuringOwn allows to set deviceID of owned device and other staff over owner TLS.
// returns new deviceID
func WithActionDuringOwn(actionDuringOwn ActionDuringOwnFunc) OwnOption {
	return func(o ownCfg) ownCfg {
		o.actionDuringOwn = actionDuringOwn
		return o
	}
}

type connUpdateResourcer interface {
	UpdateResource(context.Context, string, interface{}, interface{}, ...kitNetCoap.OptionFunc) error
}

func disown(ctx context.Context, conn connUpdateResourcer) error {
	deadline, ok := ctx.Deadline()
	if !ok || deadline.Sub(time.Now()) < time.Second {
		ctx1, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		ctx = ctx1

	}
	setResetProvisionState := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RESET,
		},
	}
	return conn.UpdateResource(ctx, "/oic/sec/pstat", setResetProvisionState, nil)
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

	return MakeNotFound(fmt.Errorf("device not found"))
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
			return MakeInternalStr("cannot connect to "+coapAddr.URL()+" for select OTM: %w", err)
		}
		defer coapConn.Close()
		return setOTM(ctx, coapConn, selectOwnerTransferMethod)
	}
	return d.selectOTMViaDiscovery(ctx, selectOwnerTransferMethod)
}

func (d *Device) setACL(ctx context.Context, links schema.ResourceLinks, ownerID string) error {
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
	setACL := acl.UpdateRequest{
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

	return d.UpdateResource(ctx, link, setACL, nil)
}

// Own set ownership of device
func (d *Device) Own(
	ctx context.Context,
	links schema.ResourceLinks,
	otmClient OTMClient,
	options ...OwnOption,
) error {
	cfg := ownCfg{
		actionDuringOwn: func(ctx context.Context, client *kitNetCoap.ClientCloseHandler) (string, error) {
			setDeviceOwned := schema.DoxmUpdate{
				DeviceID: d.DeviceID(),
			}
			/*doxm doesn't send any content for select OTM*/
			err := client.UpdateResource(ctx, "/oic/sec/doxm", setDeviceOwned, nil)
			if err != nil {
				return "", MakeInternal(fmt.Errorf("cannot set device id %v for owned device: %w", d.DeviceID(), err))
			}
			return d.DeviceID(), nil
		},
	}
	const errMsg = "cannot own device: %w"
	for _, opt := range options {
		cfg = opt(cfg)
	}

	ownership, err := d.GetOwnership(ctx)
	if err != nil {
		return MakeUnavailable(err)
	}

	sdkID, err := d.GetSdkOwnerID()
	if err != nil {
		return MakeUnavailable(fmt.Errorf("cannot set device owner: %w", err))
	}

	if ownership.Owned {
		if ownership.OwnerID == sdkID {
			return nil
		}
		return MakePermissionDenied(fmt.Errorf("device is already owned by %v", ownership.OwnerID))
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
		return MakeUnavailable(fmt.Errorf("ownership transfer method '%v' is unsupported, supported are: %v", otmClient.Type(), ownership.SupportedOwnerTransferMethods))
	}

	err = d.selectOTM(ctx, otmClient.Type(), links)
	if err != nil {
		return MakeInternal(fmt.Errorf("cannot select otm: %w", err))
	}

	var tlsClient *kitNetCoap.ClientCloseHandler
	var errors []error
	var secureEndpoints []schema.Endpoint
	for _, link := range links {
		if addr, err := link.GetUDPSecureAddr(); err == nil {
			tlsClient, err = otmClient.Dial(ctx, addr)
			if err == nil {
				secureEndpoints = append(secureEndpoints, schema.Endpoint{URI: addr.URL()})
				addr, err = link.GetTCPSecureAddr()
				if err == nil {
					secureEndpoints = append(secureEndpoints, schema.Endpoint{URI: addr.URL()})
				}
				break
			}
			errors = append(errors, fmt.Errorf("cannot connect to %v: %w", addr.URL(), err))
		}
		if addr, err := link.GetTCPSecureAddr(); err == nil {
			tlsClient, err = otmClient.Dial(ctx, addr)
			if err == nil {
				secureEndpoints = append(secureEndpoints, schema.Endpoint{URI: addr.URL()})
				addr, err = link.GetUDPSecureAddr()
				if err == nil {
					secureEndpoints = append(secureEndpoints, schema.Endpoint{URI: addr.URL()})
				}
				break
			}
			errors = append(errors, fmt.Errorf("cannot connect to %v: %w", addr.URL(), err))
		}
	}
	if tlsClient == nil {
		if len(errors) == 0 {
			return MakeInternal(fmt.Errorf("cannot get udp/tcp secure address: not found"))
		}
		return MakeInternal(fmt.Errorf("cannot get udp/tcp secure address: %+v", errors))
	}
	defer tlsClient.Close()

	var provisionState schema.ProvisionStatusResponse
	err = tlsClient.GetResource(ctx, "/oic/sec/pstat", &provisionState)
	if err != nil {
		return MakeInternal(fmt.Errorf("cannot get provision state %w", err))
	}

	if provisionState.DeviceOnboardingState.Pending {
		return MakeInternal(fmt.Errorf("device pending for operation state %v", provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState))
	}

	if provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState != schema.OperationalState_RFOTM {
		return MakeInternal(fmt.Errorf("device operation state %v is not %v", provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState, schema.OperationalState_RFOTM))
	}

	if !provisionState.SupportedOperationalModes.Has(schema.OperationalMode_CLIENT_DIRECTED) {
		return MakeUnavailable(fmt.Errorf("device supports %v, but only %v is supported", provisionState.SupportedOperationalModes, schema.OperationalMode_CLIENT_DIRECTED))
	}

	updateProvisionState := schema.ProvisionStatusUpdateRequest{
		CurrentOperationalMode: schema.OperationalMode_CLIENT_DIRECTED,
	}
	/*pstat doesn't send any content for select OperationalMode*/
	err = tlsClient.UpdateResource(ctx, "/oic/sec/pstat", updateProvisionState, nil)
	if err != nil {
		return MakeInternal(fmt.Errorf("cannot update provision state %w", err))
	}

	if cfg.actionDuringOwn != nil {
		deviceID, err := cfg.actionDuringOwn(ctx, tlsClient)
		if err != nil {
			return err
		}
		d.setDeviceID(deviceID)
	}

	/*setup credentials */
	err = otmClient.ProvisionOwnerCredentials(ctx, tlsClient, sdkID, d.DeviceID())
	if err != nil {
		return MakeAborted(fmt.Errorf("cannot provision owner %w", err))
	}

	setDeviceOwner := schema.DoxmUpdate{
		OwnerID: sdkID,
	}

	/*doxm doesn't send any content for select OTM*/
	err = tlsClient.UpdateResource(ctx, "/oic/sec/doxm", setDeviceOwner, nil)
	if err != nil {
		return MakeUnavailable(fmt.Errorf("cannot set device owner %w", err))
	}

	/*verify ownership*/
	var verifyOwner schema.Doxm
	err = tlsClient.GetResource(ctx, "/oic/sec/doxm", &verifyOwner)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(fmt.Errorf("cannot disown device: %w", errDisown))
		}
		return MakeUnavailable(fmt.Errorf("cannot verify owner %w", err))
	}
	if verifyOwner.OwnerID != sdkID {
		return MakeInternal(err)
	}

	setDeviceOwned := schema.DoxmUpdate{
		ResourceOwner: sdkID,
		Owned:         true,
	}

	/*pstat set owner of resource*/
	setOwnerProvisionState := schema.ProvisionStatusUpdateRequest{
		ResourceOwner: sdkID,
	}
	err = tlsClient.UpdateResource(ctx, "/oic/sec/pstat", setOwnerProvisionState, nil)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(fmt.Errorf("cannot disown device: %w", errDisown))
		}
		return MakeInternal(fmt.Errorf("cannot set owner of resource pstat %w", err))
	}

	/*acl2 set owner of resource*/
	setOwnerACL := acl.UpdateRequest{
		ResourceOwner: sdkID,
	}
	err = tlsClient.UpdateResource(ctx, "/oic/sec/acl2", setOwnerACL, nil)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(fmt.Errorf("cannot disown device: %w", errDisown))
		}
		return MakeInternal(fmt.Errorf("cannot set owner of resource acl2: %w", err))
	}

	/*doxm doesn't send any content for select OTM*/
	err = tlsClient.UpdateResource(ctx, "/oic/sec/doxm", setDeviceOwned, nil)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(fmt.Errorf("cannot disown device: %w", errDisown))
		}
		return MakeInternal(fmt.Errorf("cannot set device owned %w", err))
	}

	/*set device to provision opertaion mode*/
	provisionOperationState := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RFPRO,
		},
	}

	err = tlsClient.UpdateResource(ctx, "/oic/sec/pstat", provisionOperationState, nil)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(fmt.Errorf("cannot disown device: %w", errDisown))
		}
		return MakeInternal(fmt.Errorf("cannot set device to provision operation mode: %w", err))
	}

	//For Servers based on OCF 1.0, PostOwnerAcl can be executed using
	//the already-existing session. However, get ready here to use the
	//Owner Credential for establishing future secure sessions.
	//
	//For Servers based on OIC 1.1, PostOwnerAcl might fail with status
	//OC_STACK_UNAUTHORIZED_REQ. After such a failure, OwnerAclHandler
	//will close the current session and re-establish a new session,
	//using the Owner Credential.

	links, err = d.GetResourceLinks(ctx, secureEndpoints)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(fmt.Errorf("cannot disown device: %w", errDisown))
		}
		return MakeUnavailable(fmt.Errorf("cannot get resource links: %w", err))
	}

	/*set owner acl*/
	err = d.setACL(ctx, links, sdkID)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(fmt.Errorf("cannot disown device: %w", errDisown))
		}
		return MakeInternal(fmt.Errorf("cannot update resource acl: %w", err))
	}

	// Provision the device to switch back to normal operation.
	p, err := d.Provision(ctx, links)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(fmt.Errorf("cannot disown device: %w", errDisown))
		}
		return fmt.Errorf(errMsg, err)
	}

	err = p.Close(ctx)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(fmt.Errorf("cannot disown device: %w", errDisown))
		}
		return fmt.Errorf(errMsg, err)
	}
	d.Close(ctx)

	return nil
}
