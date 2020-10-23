package local

import (
	"context"
	"fmt"

	"github.com/plgd-dev/sdk/local/core"
	"github.com/plgd-dev/sdk/schema/cloud"

	"github.com/plgd-dev/sdk/schema"
	"github.com/plgd-dev/sdk/schema/acl"
)

func setACLForCloudResources(ctx context.Context, p *core.ProvisioningClient, links schema.ResourceLinks) error {
	link, err := core.GetResourceLink(links, "/oic/sec/acl2")
	if err != nil {
		return err
	}
	ownerID, err := p.GetSdkOwnerID()
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

func configureDeviceInProvsion(ctx context.Context, d *RefDevice, links schema.ResourceLinks) (rerr error) {
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

func (c *Client) OwnDevice(ctx context.Context, deviceID string, opts ...OwnOption) (string, error) {
	cfg := ownOptions{
		otmType: OTMType_Manufacturer,
	}
	for _, o := range opts {
		cfg = o.applyOnOwn(cfg)
	}
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return "", err
	}
	defer d.Release(ctx)
	ok, err := d.IsSecured(ctx, links)
	if err != nil {
		return "", err
	}
	if !ok {
		// don't own insecure device
		return deviceID, nil
	}

	return c.deviceOwner.OwnDevice(ctx, deviceID, cfg.otmType, c.ownDeviceWithSigners, cfg.opts...)
}

func (c *Client) ownDeviceWithSigners(ctx context.Context, deviceID string, otmClient core.OTMClient, opts ...core.OwnOption) (string, error) {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return "", err
	}
	defer d.Release(ctx)
	ok, err := d.IsSecured(ctx, links)
	if err != nil {
		return "", err
	}
	if !ok {
		// don't own insecure device
		return d.DeviceID(), nil
	}

	err = d.Own(ctx, links, otmClient, opts...)
	if err != nil {
		return "", err
	}

	err = configureDeviceInProvsion(ctx, d, links)
	if err != nil {
		d.Disown(ctx, links)
		return "", err
	}

	return d.DeviceID(), nil
}
