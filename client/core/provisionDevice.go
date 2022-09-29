package core

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/schema/pstat"
	"github.com/plgd-dev/kit/v2/strings"
)

func (d *Device) Provision(ctx context.Context, links schema.ResourceLinks) (*ProvisioningClient, error) {
	p := ProvisioningClient{
		Device: d,
		links:  links,
	}
	err := p.start(ctx)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type ProvisioningClient struct {
	*Device
	links schema.ResourceLinks
}

func (c *ProvisioningClient) start(ctx context.Context) error {
	provisioningState := pstat.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &pstat.DeviceOnboardingState{
			CurrentOrPendingOperationalState: pstat.OperationalState_RFPRO,
		},
	}
	const errMsg = "could not start provisioning the device: %w"
	link, err := GetResourceLink(c.links, pstat.ResourceURI)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	link.Endpoints = link.GetSecureEndpoints()

	err = c.UpdateResource(ctx, link, provisioningState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot update to provisioning state %+v: %w", link, err))
	}
	return nil
}

func (c *ProvisioningClient) Close(ctx context.Context) error {
	normalOperationState := pstat.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &pstat.DeviceOnboardingState{
			CurrentOrPendingOperationalState: pstat.OperationalState_RFNOP,
		},
	}
	const errMsg = "could not finalize provisioning the device: %w"
	link, err := GetResourceLink(c.links, pstat.ResourceURI)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	link.Endpoints = link.GetSecureEndpoints()

	err = c.UpdateResource(ctx, link, normalOperationState, nil)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

func (c *ProvisioningClient) AddCredentials(ctx context.Context, cr credential.CredentialUpdateRequest) error {
	const errMsg = "could not add credentials to the device: %w"
	link, err := GetResourceLink(c.links, credential.ResourceURI)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	link.Endpoints = link.GetSecureEndpoints()
	err = c.UpdateResource(ctx, link, cr, nil)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

func (c *ProvisioningClient) AddCertificateAuthority(ctx context.Context, subject string, cert *x509.Certificate) error {
	setCaCredential := credential.CredentialUpdateRequest{
		Credentials: []credential.Credential{
			{
				Subject: subject,
				Type:    credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
				Usage:   credential.CredentialUsage_TRUST_CA,
				PublicData: &credential.CredentialPublicData{
					DataInternal: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})),
					Encoding:     credential.CredentialPublicDataEncoding_PEM,
				},
			},
		},
	}
	return c.AddCredentials(ctx, setCaCredential)
}

func (c *ProvisioningClient) SetCloudResource(ctx context.Context, r cloud.ConfigurationUpdateRequest) error {
	switch {
	case r.AuthorizationProvider == "":
		return fmt.Errorf("invalid AuthorizationProvider")
	case r.AuthorizationCode == "":
		return fmt.Errorf("invalid AuthorizationCode")
	case r.URL == "":
		return fmt.Errorf("invalid URL")
	}
	var link schema.ResourceLink

	for _, l := range c.links {
		if strings.SliceContains(l.ResourceTypes, cloud.ConfigurationResourceType) {
			link = l
			break
		}
	}
	if link.Href == "" {
		return fmt.Errorf("could not resolve cloud resource link of device %s", c.DeviceID())
	}
	link.Endpoints = link.GetSecureEndpoints()
	err := c.UpdateResource(ctx, link, r, nil)
	if err != nil {
		return fmt.Errorf("could not set cloud resource of device %s: %w", c.DeviceID(), err)
	}
	return nil
}

// Usage: SetAccessControl(ctx, schema.AllPermissions, schema.TLSConnection, schema.AllResources)
func (c *ProvisioningClient) SetAccessControl(
	ctx context.Context,
	permission acl.Permission,
	subject acl.Subject,
	resources ...acl.Resource,
) error {
	setACL := acl.UpdateRequest{
		AccessControlList: []acl.AccessControl{
			{
				Permission: permission,
				Subject:    subject,
				Resources:  resources,
			},
		},
	}
	const errMsg = "could not update ACL of the device: %w"
	link, err := GetResourceLink(c.links, acl.ResourceURI)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	link.Endpoints = link.GetSecureEndpoints()
	err = c.UpdateResource(ctx, link, setACL, nil)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}
