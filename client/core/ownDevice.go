package core

import (
	"context"
	"fmt"
	"time"

	"github.com/plgd-dev/device/client/core/otm"
	kitNetCoap "github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/acl"
	"github.com/plgd-dev/device/schema/cloud"
	"github.com/plgd-dev/device/schema/csr"
	"github.com/plgd-dev/device/schema/device"
	"github.com/plgd-dev/device/schema/doxm"
	"github.com/plgd-dev/device/schema/maintenance"
	"github.com/plgd-dev/device/schema/platform"
	"github.com/plgd-dev/device/schema/pstat"
	"github.com/plgd-dev/device/schema/resources"
	"github.com/plgd-dev/device/schema/sdi"
	"github.com/plgd-dev/device/schema/sp"
	kitNet "github.com/plgd-dev/kit/v2/net"
)

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
	if !ok || time.Until(deadline) < time.Second {
		ctx1, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		ctx = ctx1

	}
	setResetProvisionState := pstat.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &pstat.DeviceOnboardingState{
			CurrentOrPendingOperationalState: pstat.OperationalState_RESET,
		},
	}
	return conn.UpdateResource(ctx, pstat.ResourceURI, setResetProvisionState, nil)
}

func setOTM(ctx context.Context, conn connUpdateResourcer, selectOwnerTransferMethod doxm.OwnerTransferMethod) error {
	selectOTM := doxm.DoxmUpdate{
		SelectOwnerTransferMethod: &selectOwnerTransferMethod,
	}
	/*doxm doesn't send any content for update*/
	return conn.UpdateResource(ctx, doxm.ResourceURI, selectOTM, nil)
}

func (d *Device) selectOTM(ctx context.Context, selectOwnerTransferMethod doxm.OwnerTransferMethod) error {
	endpoints := d.GetEndpoints()
	coapAddr, err := endpoints.GetAddr(schema.UDPScheme)
	if err != nil {
		return err
	}
	coapConn, err := kitNetCoap.DialUDP(ctx, coapAddr.String())
	if err != nil {
		return MakeInternalStr("cannot connect to "+coapAddr.URL()+" for select OTM: %w", err)
	}
	defer coapConn.Close()
	return setOTM(ctx, coapConn, selectOwnerTransferMethod)
}

func (d *Device) setACL(ctx context.Context, links schema.ResourceLinks, ownerID string) error {
	link, err := GetResourceLink(links, acl.ResourceURI)
	if err != nil {
		return err
	}

	// CleanUp acls rules
	err = d.DeleteResource(ctx, link, nil)
	if err != nil {
		return err
	}

	confResources := make([]acl.Resource, 0, 1)
	for _, href := range links.GetResourceHrefs(cloud.ConfigurationResourceType) {
		confResources = append(confResources, acl.Resource{
			Href:       href,
			Interfaces: []string{"*"},
		})
	}
	for _, href := range links.GetResourceHrefs(maintenance.ResourceType) {
		confResources = append(confResources, acl.Resource{
			Href:       href,
			Interfaces: []string{"*"},
		})
	}

	/*acl2 set owner of resource*/
	setACL := acl.UpdateRequest{
		AccessControlList: []acl.AccessControl{
			{
				Permission: acl.AllPermissions,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: ownerID,
					},
				},
				Resources: acl.AllResources,
			},
			{
				Permission: acl.AllPermissions,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: ownerID,
					},
				},
				Resources: confResources,
			},
			{
				Permission: acl.Permission_READ | acl.Permission_WRITE | acl.Permission_DELETE,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: ownerID,
					},
				},
				Resources: []acl.Resource{
					{
						Href:       sp.ResourceURI,
						Interfaces: []string{"*"},
					},
					{
						Href:       pstat.ResourceURI,
						Interfaces: []string{"*"},
					},
					{
						Href:       doxm.ResourceURI,
						Interfaces: []string{"*"},
					},
				},
			},
			{
				Permission: acl.Permission_READ,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: ownerID,
					},
				},
				Resources: []acl.Resource{
					{
						Href:       csr.ResourceURI,
						Interfaces: []string{"*"},
					},
				},
			},
			{
				Permission: acl.Permission_READ,
				Subject: acl.Subject{
					Subject_Connection: &acl.Subject_Connection{
						Type: acl.ConnectionType_ANON_CLEAR,
					},
				},
				Resources: []acl.Resource{
					{
						Href:       device.ResourceURI,
						Interfaces: []string{"*"},
					},
					{
						Href:       platform.ResourceURI,
						Interfaces: []string{"*"},
					},
					{
						Href:       resources.ResourceURI,
						Interfaces: []string{"*"},
					},
					{
						Href:       sdi.ResourceURI,
						Interfaces: []string{"*"},
					},
					{
						Href:       doxm.ResourceURI,
						Interfaces: []string{"*"},
					},
				},
			},
		},
	}

	return d.UpdateResource(ctx, link, setACL, nil)
}

func disownError(err error) error {
	return fmt.Errorf("cannot disown device: %w", err)
}

// Own set ownership of device
func (d *Device) Own(
	ctx context.Context,
	links schema.ResourceLinks,
	otmClient otm.Client,
	options ...OwnOption,
) error {
	cfg := ownCfg{
		actionDuringOwn: func(ctx context.Context, client *kitNetCoap.ClientCloseHandler) (string, error) {
			deviceID := d.DeviceID()
			setDeviceOwned := doxm.DoxmUpdate{
				DeviceID: &deviceID,
			}
			/*doxm doesn't send any content for select OTM*/
			err := client.UpdateResource(ctx, doxm.ResourceURI, setDeviceOwned, nil)
			if err != nil {
				return "", MakeInternal(fmt.Errorf("cannot set device id %v for owned device: %w", deviceID, err))
			}
			return deviceID, nil
		},
	}
	const errMsg = "cannot own device: %w"
	for _, opt := range options {
		cfg = opt(cfg)
	}

	ownership, err := d.GetOwnership(ctx, links)
	if err != nil {
		return MakeUnavailable(fmt.Errorf("cannot get ownership: %w", err))
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

	err = d.selectOTM(ctx, otmClient.Type())
	if err != nil {
		return MakeInternal(fmt.Errorf("cannot select otm: %w", err))
	}
	var tlsClient *kitNetCoap.ClientCloseHandler
	var tlsAddr kitNet.Addr
	var errors []error
	for _, link := range links {
		if addr, err := link.GetUDPSecureAddr(); err == nil {
			tlsClient, err = otmClient.Dial(ctx, addr)
			if err == nil {
				tlsAddr = addr
				break
			}
			errors = append(errors, fmt.Errorf("cannot connect to %v: %w", addr.URL(), err))
		}
		if addr, err := link.GetTCPSecureAddr(); err == nil {
			tlsClient, err = otmClient.Dial(ctx, addr)
			if err == nil {
				tlsAddr = addr
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

	var provisionState pstat.ProvisionStatusResponse
	err = tlsClient.GetResource(ctx, pstat.ResourceURI, &provisionState)
	if err != nil {
		return MakeInternal(fmt.Errorf("cannot get provision state %w", err))
	}

	if provisionState.DeviceOnboardingState.Pending {
		return MakeInternal(fmt.Errorf("device pending for operation state %v", provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState))
	}

	if provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState != pstat.OperationalState_RFOTM {
		return MakeInternal(fmt.Errorf("device operation state %v is not %v", provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState, pstat.OperationalState_RFOTM))
	}

	if !provisionState.SupportedOperationalModes.Has(pstat.OperationalMode_CLIENT_DIRECTED) {
		return MakeUnavailable(fmt.Errorf("device supports %v, but only %v is supported", provisionState.SupportedOperationalModes, pstat.OperationalMode_CLIENT_DIRECTED))
	}

	updateProvisionState := pstat.ProvisionStatusUpdateRequest{
		CurrentOperationalMode: pstat.OperationalMode_CLIENT_DIRECTED,
	}
	/*pstat doesn't send any content for select OperationalMode*/
	err = tlsClient.UpdateResource(ctx, pstat.ResourceURI, updateProvisionState, nil)
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

	setDeviceOwner := doxm.DoxmUpdate{
		OwnerID: &sdkID,
	}

	/*doxm doesn't send any content for select OTM*/
	err = tlsClient.UpdateResource(ctx, doxm.ResourceURI, setDeviceOwner, nil)
	if err != nil {
		return MakeUnavailable(fmt.Errorf("cannot set device owner %w", err))
	}

	/*verify ownership*/
	var verifyOwner doxm.Doxm
	err = tlsClient.GetResource(ctx, doxm.ResourceURI, &verifyOwner)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(disownError(errDisown))
		}
		return MakeUnavailable(fmt.Errorf("cannot verify owner %w", err))
	}
	if verifyOwner.OwnerID != sdkID {
		return MakeInternal(err)
	}

	owned := true
	setDeviceOwned := doxm.DoxmUpdate{
		ResourceOwner: &sdkID,
		Owned:         &owned,
	}

	/*pstat set owner of resource*/
	setOwnerProvisionState := pstat.ProvisionStatusUpdateRequest{
		ResourceOwner: sdkID,
	}
	err = tlsClient.UpdateResource(ctx, pstat.ResourceURI, setOwnerProvisionState, nil)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(disownError(errDisown))
		}
		return MakeInternal(fmt.Errorf("cannot set owner of resource pstat %w", err))
	}

	/*acl2 set owner of resource*/
	setOwnerACL := acl.UpdateRequest{
		ResourceOwner: sdkID,
	}
	err = tlsClient.UpdateResource(ctx, acl.ResourceURI, setOwnerACL, nil)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(disownError(errDisown))
		}
		return MakeInternal(fmt.Errorf("cannot set owner of resource acl2: %w", err))
	}

	/*doxm doesn't send any content for select OTM*/
	err = tlsClient.UpdateResource(ctx, doxm.ResourceURI, setDeviceOwned, nil)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(disownError(errDisown))
		}
		return MakeInternal(fmt.Errorf("cannot set device owned %w", err))
	}

	/*set device to provision opertaion mode*/
	provisionOperationState := pstat.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &pstat.DeviceOnboardingState{
			CurrentOrPendingOperationalState: pstat.OperationalState_RFPRO,
		},
	}

	err = tlsClient.UpdateResource(ctx, pstat.ResourceURI, provisionOperationState, nil)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(disownError(errDisown))
		}
		return MakeInternal(fmt.Errorf("cannot set device to provision operation mode: %w", err))
	}

	links, err = getResourceLinks(ctx, tlsAddr, tlsClient, d.GetEndpoints())
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(disownError(errDisown))
		}
		return MakeUnavailable(fmt.Errorf("cannot get resource links: %w", err))
	}

	/*set owner acl*/
	err = d.setACL(ctx, links, sdkID)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(disownError(errDisown))
		}
		return MakeInternal(fmt.Errorf("cannot update resource acl: %w", err))
	}

	// Provision the device to switch back to normal operation.
	p, err := d.Provision(ctx, links)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(disownError(errDisown))
		}
		return fmt.Errorf(errMsg, err)
	}
	err = p.Close(ctx)
	if err != nil {
		if errDisown := disown(ctx, tlsClient); errDisown != nil {
			d.cfg.errFunc(disownError(errDisown))
		}
		return fmt.Errorf(errMsg, err)
	}
	return nil
}
