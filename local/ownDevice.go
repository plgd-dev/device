package local

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/go-ocf/sdk/schema/cloud"

	ocf "github.com/go-ocf/sdk/local/core"
	ocfSchema "github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/schema/acl"
)

func setACLForCloudResources(ctx context.Context, p *ocf.ProvisioningClient, links ocfSchema.ResourceLinks) error {
	link, err := ocf.GetResourceLink(links, "/oic/sec/acl2")
	if err != nil {
		return err
	}
	ownerID, err := p.GetSdkDeviceID()
	if err != nil {
		return err
	}

	aclResources := make([]acl.Resource, 0, 1)
	for _, res := range links.GetResourceLinks(cloud.ConfigurationResourceType) {
		aclResources = append(aclResources, acl.Resource{
			Href:       res.Href,
			Interfaces: []string{"*"},
		})
	}
	if len(aclResources) == 0 {
		return fmt.Errorf("cannot find %v resource", cloud.ConfigurationResourceType)
	}

	obACL := acl.UpdateRequest{
		AccessControlList: []acl.AccessControl{
			acl.AccessControl{
				Permission: acl.AllPermissions,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: ownerID,
					},
				},
				Resources: aclResources,
			},
		},
	}

	return p.UpdateResource(ctx, link, obACL, nil)
}

func configureDeviceInProvsion(ctx context.Context, d *refDevice, links ocfSchema.ResourceLinks, certAuthorities []*x509.Certificate) (rerr error) {
	p, err := d.Provision(ctx, links)
	if err != nil {
		return err
	}
	defer func() {
		err := p.Close(ctx)
		if err != nil && rerr == nil {
			rerr = err
		}
	}()

	err = setACLForCloudResources(ctx, p, links)
	if err != nil {
		return fmt.Errorf("cannot set acl for cloud resources: %w", err)
	}
	return nil
}

func (c *Client) OwnDevice(ctx context.Context, deviceID string, opts ...ocf.OwnOption) error {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)
	ok, err := d.IsSecured(ctx, links)
	if err != nil {
		return err
	}
	if !ok {
		// don't own insecure device
		return nil
	}

	return c.deviceOwner.OwnDevice(ctx, deviceID, c.ownDeviceWithSigners, opts...)
}

func (c *Client) ownDeviceWithSigners(ctx context.Context, deviceID string, otmClient ocf.OTMClient, opts ...ocf.OwnOption) error {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)
	ok, err := d.IsSecured(ctx, links)
	if err != nil {
		return err
	}
	if !ok {
		// don't own insecure device
		return nil
	}

	certAuthorities, err := c.app.GetRootCertificateAuthorities()
	if err != nil {
		return err
	}

	err = d.Own(ctx, links, otmClient, opts...)
	if err != nil {
		return err
	}

	err = configureDeviceInProvsion(ctx, d, links, certAuthorities)
	if err != nil {
		d.Disown(ctx, links)
		return err
	}

	return nil
}
