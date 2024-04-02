// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package core

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/pion/dtls/v2"
	"github.com/plgd-dev/device/v2/client/core/otm"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/csr"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/schema/pstat"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/device/v2/schema/sdi"
	"github.com/plgd-dev/device/v2/schema/softwareupdate"
	"github.com/plgd-dev/device/v2/schema/sp"
	kitNet "github.com/plgd-dev/kit/v2/net"
)

type (
	ActionDuringOwnFunc = func(ctx context.Context, client *coap.ClientCloseHandler) (string, error)
	ActionAfterOwnFunc  = func(ctx context.Context, client *coap.ClientCloseHandler) error
)

type ownCfg struct {
	psk             []byte
	sign            otm.SignFunc
	actionDuringOwn ActionDuringOwnFunc
	actionAfterOwn  ActionAfterOwnFunc
}

type OwnOption = func(ownCfg) ownCfg

// WithActionDuringOwn allows to set deviceID of owned device and other staff over owner TLS.
// returns new deviceID, if it returns error device will be disowned.
func WithActionDuringOwn(actionDuringOwn ActionDuringOwnFunc) OwnOption {
	return func(o ownCfg) ownCfg {
		o.actionDuringOwn = actionDuringOwn
		return o
	}
}

// WithActionAfterOwn allows initialize configuration at the device via DTLS connection with preshared key. For example setup time / NTP.
// if it returns error device will be disowned.
func WithActionAfterOwn(actionAfterOwn ActionAfterOwnFunc) OwnOption {
	return func(o ownCfg) ownCfg {
		o.actionAfterOwn = actionAfterOwn
		return o
	}
}

// WithPresharedKey allows to set preshared key for owner. It is not set, it will be randomized.
func WithPresharedKey(psk []byte) OwnOption {
	return func(o ownCfg) ownCfg {
		o.psk = psk
		return o
	}
}

// WithSetupCertificates signs identity ceriticates and install root ca.
func WithSetupCertificates(sign otm.SignFunc) OwnOption {
	return func(o ownCfg) ownCfg {
		o.sign = sign
		return o
	}
}

type connUpdateResourcer interface {
	UpdateResource(context.Context, string, interface{}, interface{}, ...coap.OptionFunc) error
	DeleteResource(context.Context, string, interface{}, ...coap.OptionFunc) error
}

func updateOperationalState(ctx context.Context, conn connUpdateResourcer, newState pstat.OperationalState) error {
	updateProvisionState := pstat.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &pstat.DeviceOnboardingState{
			CurrentOrPendingOperationalState: newState,
		},
	}
	return conn.UpdateResource(ctx, pstat.ResourceURI, updateProvisionState, nil)
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
	coapConn, err := coap.DialUDP(ctx, coapAddr.String())
	if err != nil {
		return MakeInternalStr("cannot connect to "+coapAddr.URL()+" for select OTM: %w", err)
	}
	defer func() {
		if errC := coapConn.Close(); errC != nil {
			d.cfg.Logger.Warn(fmt.Errorf("select otm: cannot close connection: %w", errC).Error())
		}
	}()
	return setOTM(ctx, coapConn, selectOwnerTransferMethod)
}

func setACL(ctx context.Context, conn connUpdateResourcer, links schema.ResourceLinks, ownerID string) error {
	link, err := GetResourceLink(links, acl.ResourceURI)
	if err != nil {
		return err
	}

	// CleanUp acls rules
	err = conn.DeleteResource(ctx, link.Href, nil)
	if err != nil {
		return err
	}

	confResources := acl.AllResources
	for _, href := range links.GetResourceHrefs(cloud.ResourceType) {
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
	for _, href := range links.GetResourceHrefs(softwareupdate.ResourceType) {
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

	return conn.UpdateResource(ctx, link.Href, setACL, nil)
}

// findOTMClient finds supported client in order as user wants. The first match will be used.
func findOTMClient(otmClients []otm.Client, deviceSupportedOwnerTransferMethods []doxm.OwnerTransferMethod) otm.Client {
	for _, c := range otmClients {
		for _, s := range deviceSupportedOwnerTransferMethods {
			if s == c.Type() {
				return c
			}
		}
	}
	return nil
}

func supportedOTMTypes(otmClients []otm.Client) []string {
	v := make([]string, 0, len(otmClients))
	for _, c := range otmClients {
		v = append(v, c.Type().String())
	}
	return v
}

func (d *Device) setDoxmDeviceID(ctx context.Context, cc *coap.ClientCloseHandler) (string, error) {
	deviceID := d.DeviceID()
	setDeviceOwned := doxm.DoxmUpdate{
		DeviceID: &deviceID,
	}
	/*doxm doesn't send any content for select OTM*/
	err := cc.UpdateResource(ctx, doxm.ResourceURI, setDeviceOwned, nil)
	if err != nil {
		return "", MakeInternal(fmt.Errorf("cannot set device id %v for owned device: %w", deviceID, err))
	}
	return deviceID, nil
}

func getTLSClient(ctx context.Context, links schema.ResourceLinks, otmClient otm.Client) (*coap.ClientCloseHandler, kitNet.Addr, error) {
	var errs *multierror.Error
	for _, link := range links {
		if addr, err := link.GetUDPSecureAddr(); err == nil {
			tlsClient, err := otmClient.Dial(ctx, addr)
			if err == nil {
				return tlsClient, addr, nil
			}
			errs = multierror.Append(errs, fmt.Errorf("cannot connect to %v: %w", addr.URL(), err))
		}
		if addr, err := link.GetTCPSecureAddr(); err == nil {
			tlsClient, err := otmClient.Dial(ctx, addr)
			if err == nil {
				return tlsClient, addr, nil
			}
			errs = multierror.Append(errs, fmt.Errorf("cannot connect to %v: %w", addr.URL(), err))
		}
	}
	if errs.ErrorOrNil() != nil {
		return nil, kitNet.Addr{}, errs
	}
	return nil, kitNet.Addr{}, errors.New("not found")
}

func disown(ctx context.Context, conn connUpdateResourcer) error {
	deadline, ok := ctx.Deadline()
	if ctx.Err() != nil || !ok || time.Until(deadline) < time.Second {
		ctx1, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		ctx = ctx1
	}
	return updateOperationalState(ctx, conn, pstat.OperationalState_RESET)
}

func (d *Device) disownAndLogError(ctx context.Context, conn connUpdateResourcer) {
	if err := disown(ctx, conn); err != nil {
		d.cfg.Logger.Warn(fmt.Errorf("cannot disown device: %w", err).Error())
	}
}

func otmErrorf(oc otm.Client, format string, a ...any) error {
	return fmt.Errorf("otmClient: %v: %w", oc.Type(), fmt.Errorf(format, a...))
}

func checkProvisionState(ctx context.Context, cc *coap.ClientCloseHandler, oc otm.Client) error {
	errorf := func(format string, a ...any) error {
		return otmErrorf(oc, format, a...)
	}

	var provisionState pstat.ProvisionStatusResponse
	err := cc.GetResource(ctx, pstat.ResourceURI, &provisionState)
	if err != nil {
		return MakeInternal(errorf("cannot get provision state %w", err))
	}

	if provisionState.DeviceOnboardingState.Pending {
		return MakeInternal(errorf("device pending for operation state %v", provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState))
	}

	if provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState != pstat.OperationalState_RFOTM {
		return MakeInternal(errorf("device operation state %v is not %v", provisionState.DeviceOnboardingState.CurrentOrPendingOperationalState, pstat.OperationalState_RFOTM))
	}

	if !provisionState.SupportedOperationalModes.Has(pstat.OperationalMode_CLIENT_DIRECTED) {
		return MakeUnavailable(errorf("device supports %v, but only %v is supported", provisionState.SupportedOperationalModes, pstat.OperationalMode_CLIENT_DIRECTED))
	}

	return nil
}

func provisionOwner(ctx context.Context, cfg ownCfg, deviceID, sdkID string, cc *coap.ClientCloseHandler, oc otm.Client) ([]byte, error) {
	errorf := func(err error) error {
		return otmErrorf(oc, "cannot provision owner: %w", err)
	}

	/*setup credentials */
	if len(cfg.psk) == 0 && cfg.sign == nil {
		return nil, MakeInvalidArgument(errorf(errors.New("both preshared and signer are empty")))
	}

	psk := make([]byte, 16)
	if len(cfg.psk) > 0 {
		psk = cfg.psk
	} else {
		_, errRead := rand.Read(psk)
		if errRead != nil {
			return nil, MakeAborted(errorf(errRead))
		}
	}

	var provisionOpts []otm.ProvisionOwnerCredentialstOption
	if cfg.sign != nil {
		provisionOpts = append(provisionOpts, otm.WithSetupCertificates(deviceID, cfg.sign))
	}
	err := otm.ProvisionOwnerCredentials(ctx, cc, sdkID, psk, provisionOpts...)
	if err != nil {
		return nil, MakeAborted(errorf(err))
	}

	return psk, nil
}

func setResourceOwner(ctx context.Context, owner string, cc *coap.ClientCloseHandler) error {
	/*pstat set owner of resource*/
	setOwnerProvisionState := pstat.ProvisionStatusUpdateRequest{
		ResourceOwner: owner,
	}
	err := cc.UpdateResource(ctx, pstat.ResourceURI, setOwnerProvisionState, nil)
	if err != nil {
		return fmt.Errorf("cannot set owner of resource pstat: %w", err)
	}

	/*acl2 set owner of resource*/
	setOwnerACL := acl.UpdateRequest{
		ResourceOwner: owner,
	}
	err = cc.UpdateResource(ctx, acl.ResourceURI, setOwnerACL, nil)
	if err != nil {
		return fmt.Errorf("cannot set owner of resource acl2: %w", err)
	}

	owned := true
	setDeviceOwned := doxm.DoxmUpdate{
		ResourceOwner: &owner,
		Owned:         &owned,
	}
	/*doxm doesn't send any content for select OTM*/
	err = cc.UpdateResource(ctx, doxm.ResourceURI, setDeviceOwned, nil)
	if err != nil {
		return fmt.Errorf("cannot set device owned: %w", err)
	}
	return nil
}

func (d *Device) ownershipTransfer(ctx context.Context, cfg ownCfg, sdkID string, cc *coap.ClientCloseHandler, oc otm.Client) ([]byte, error) {
	if err := checkProvisionState(ctx, cc, oc); err != nil {
		return nil, err
	}

	errorf := func(format string, a ...any) error {
		return otmErrorf(oc, format, a...)
	}

	setCurrentOperationalMode := pstat.ProvisionStatusUpdateRequest{
		CurrentOperationalMode: pstat.OperationalMode_CLIENT_DIRECTED,
	}
	/*pstat doesn't send any content for select OperationalMode*/
	err := cc.UpdateResource(ctx, pstat.ResourceURI, setCurrentOperationalMode, nil)
	if err != nil {
		return nil, MakeInternal(errorf("cannot update provision state: %w", err))
	}

	if cfg.actionDuringOwn != nil {
		deviceID, errOwn := cfg.actionDuringOwn(ctx, cc)
		if errOwn != nil {
			return nil, errOwn
		}
		d.SetDeviceID(deviceID)
	}

	psk, err := provisionOwner(ctx, cfg, d.DeviceID(), sdkID, cc, oc)
	if err != nil {
		return nil, err
	}

	setDeviceOwner := doxm.DoxmUpdate{
		OwnerID: &sdkID,
	}
	/*doxm doesn't send any content for select OTM*/
	err = cc.UpdateResource(ctx, doxm.ResourceURI, setDeviceOwner, nil)
	if err != nil {
		return nil, MakeUnavailable(errorf("cannot set device owner: %w", err))
	}
	/*verify ownership*/
	var verifyOwner doxm.Doxm
	err = cc.GetResource(ctx, doxm.ResourceURI, &verifyOwner)
	if err != nil {
		return nil, MakeUnavailable(errorf("cannot verify owner: %w", err))
	}
	if verifyOwner.OwnerID != sdkID {
		return nil, MakeInternal(errorf("invalid ownerID"))
	}

	err = setResourceOwner(ctx, sdkID, cc)
	if err != nil {
		return nil, MakeInternal(err)
	}

	return psk, nil
}

func getDTLSClient(ctx context.Context, psk []byte, sdkID, addr string, oc otm.Client) (*coap.ClientCloseHandler, error) {
	id, err := uuid.Parse(sdkID)
	if err != nil {
		return nil, MakeInternal(otmErrorf(oc, "invalid sdkID %v: %w", sdkID, err))
	}
	idBin, _ := id.MarshalBinary()
	dtlsConfig := dtls.Config{
		PSKIdentityHint: idBin,
		PSK: func([]byte) ([]byte, error) {
			return psk, nil
		},
		CipherSuites: []dtls.CipherSuiteID{dtls.TLS_ECDHE_PSK_WITH_AES_128_CBC_SHA256},
	}
	pskConn, err := coap.DialUDPSecure(ctx, addr, &dtlsConfig)
	if err != nil {
		return nil, MakeUnavailable(otmErrorf(oc, "cannot create connection for finish ownership transfer: %w", err))
	}
	return pskConn, nil
}

type deviceConfigurer struct {
	tlsClient      *coap.ClientCloseHandler
	otmClient      otm.Client
	ownerID        string
	address        string
	actionAfterOwn ActionAfterOwnFunc
	err            func(error)
}

func (d deviceConfigurer) configure(ctx context.Context, links schema.ResourceLinks, psk []byte) error {
	errorf := func(format string, a ...any) error {
		return otmErrorf(d.otmClient, format, a...)
	}

	/*set device to provision operation mode*/
	err := updateOperationalState(ctx, d.tlsClient, pstat.OperationalState_RFPRO)
	if err != nil {
		return MakeInternal(errorf("cannot set device to provision operation mode: %w", err))
	}

	pskConn, err := getDTLSClient(ctx, psk, d.ownerID, d.address, d.otmClient)
	if err != nil {
		return err
	}
	defer func() {
		if errC := pskConn.Close(); errC != nil {
			d.err(fmt.Errorf("cannot close DTLS connection: %w", errC))
		}
	}()

	/*set owner acl*/
	err = setACL(ctx, pskConn, links, d.ownerID)
	if err != nil {
		return MakeInternal(errorf("cannot update resource acl: %w", err))
	}

	// Provision the device to switch back to normal operation.
	err = updateOperationalState(ctx, pskConn, pstat.OperationalState_RFNOP)
	if err != nil {
		return MakeInternal(errorf("cannot update operation state to normal mode: %w", err))
	}

	if d.actionAfterOwn != nil {
		err = d.actionAfterOwn(ctx, pskConn)
		if err != nil {
			return MakeInternal(errorf("cannot create connection for finish ownership transfer: %w", err))
		}
	}
	return nil
}

// Own set ownership of device. For owning, the first match in order of otmClients with the device will be used.
// Note: In case if the device fails before changing RFOTM the iotivity-stack invokes disown by itself. This can result
// in a state where the disown is invoked two times in a row. Once by the iotivity-stack and second time by device core.
func (d *Device) Own(
	ctx context.Context,
	links schema.ResourceLinks,
	otmClients []otm.Client,
	options ...OwnOption,
) error {
	cfg := ownCfg{
		actionDuringOwn: d.setDoxmDeviceID,
	}
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

	otmClient := findOTMClient(otmClients, ownership.SupportedOwnerTransferMethods)
	if otmClient == nil {
		return MakeUnavailable(fmt.Errorf("ownership transfer methods used by clients '%v' are not compatible with the device methods '%v'", supportedOTMTypes(otmClients), ownership.SupportedOwnerTransferMethods))
	}

	errorf := func(format string, a ...any) error {
		return otmErrorf(otmClient, format, a...)
	}

	if err = d.selectOTM(ctx, otmClient.Type()); err != nil {
		return MakeInternal(errorf("cannot select otm: %w", err))
	}

	tlsClient, tlsAddr, err := getTLSClient(ctx, links, otmClient)
	if err != nil {
		return MakeInternal(errorf("cannot get udp/tcp secure address: %v", err))
	}
	defer func() {
		if errC := tlsClient.Close(); errC != nil {
			d.cfg.Logger.Debug(fmt.Errorf("cannot close TLS connection: %w", errC).Error())
		}
	}()

	psk, err := d.ownershipTransfer(ctx, cfg, sdkID, tlsClient, otmClient)
	if err != nil {
		d.disownAndLogError(ctx, tlsClient)
		return err
	}

	dc := deviceConfigurer{
		tlsClient:      tlsClient,
		otmClient:      otmClient,
		ownerID:        sdkID,
		address:        tlsAddr.String(),
		actionAfterOwn: cfg.actionAfterOwn,
		err: func(err error) {
			d.cfg.Logger.Debug(err.Error())
		},
	}
	if err = dc.configure(ctx, links, psk); err != nil {
		d.disownAndLogError(ctx, tlsClient)
		return err
	}
	return nil
}
