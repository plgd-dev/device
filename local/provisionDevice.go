package local

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/go-ocf/kit/strings"
	"github.com/go-ocf/sdk/schema"
)

func (c *Client) ProvisionDevice(ctx context.Context, deviceID string) (*ProvisioningClient, error) {
	p := ProvisioningClient{Client: c, deviceID: deviceID}
	err := p.start(ctx)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type ProvisioningClient struct {
	*Client
	deviceID string
}

func (c *ProvisioningClient) start(ctx context.Context) error {
	provisioningState := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RFPRO,
		},
	}
	err := c.UpdateResource(ctx, c.deviceID, "/oic/sec/pstat", provisioningState, nil)
	if err != nil {
		return fmt.Errorf("could not start provisioning the device %s: %v", c.deviceID, err)
	}
	return nil
}

func (c *ProvisioningClient) Close(ctx context.Context) error {
	normalOperationState := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RFNOP,
		},
	}
	err := c.UpdateResource(ctx, c.deviceID, "/oic/sec/pstat", normalOperationState, nil)
	if err != nil {
		return fmt.Errorf("could not finalize provisioning the device %s: %v", c.deviceID, err)
	}
	return nil
}

func (c *ProvisioningClient) AddCertificateAuthority(ctx context.Context, subject string, cert *x509.Certificate) error {
	setCaCredential := schema.CredentialUpdateRequest{
		Credentials: []schema.Credential{
			schema.Credential{
				Subject: subject,
				Type:    schema.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE,
				Usage:   schema.CredentialUsage_TRUST_CA,
				PublicData: schema.CredentialPublicData{
					Data:     string(cert.Raw),
					Encoding: schema.CredentialPublicDataEncoding_DER,
				},
			},
		},
	}
	err := c.UpdateResource(ctx, c.deviceID, "/oic/sec/cred", setCaCredential, nil)
	if err != nil {
		return fmt.Errorf("could not add certificate to device %s: %v", c.deviceID, err)
	}
	return nil
}

func (c *ProvisioningClient) SetCloudResource(ctx context.Context, r schema.CloudUpdateRequest) error {
	switch {
	case r.AuthorizationProvider == "":
		return fmt.Errorf("invalid AuthorizationProvider")
	case r.AuthorizationCode == "":
		return fmt.Errorf("invalid AuthorizationCode")
	case r.URL == "":
		return fmt.Errorf("invalid URL")
	}
	var href string
	for _, l := range c.factory.GetLinks() {
		if l.GetDeviceID() == c.deviceID && strings.SliceContains(l.ResourceTypes, schema.CloudResourceType) {
			href = l.Href
			break
		}
	}
	if href == "" {
		return fmt.Errorf("could not resolve cloud resource link of device %s", c.deviceID)
	}
	err := c.UpdateResource(ctx, c.deviceID, href, r, nil)
	if err != nil {
		return fmt.Errorf("could not set cloud resource of device %s: %v", c.deviceID, err)
	}
	return nil
}
