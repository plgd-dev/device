package local

import (
	"context"
	"fmt"

	"github.com/plgd-dev/sdk/local/core"
	"github.com/plgd-dev/sdk/schema"
	"github.com/plgd-dev/sdk/schema/acl"
	"github.com/plgd-dev/sdk/schema/cloud"
)

func setCloudResource(ctx context.Context, links schema.ResourceLinks, d *RefDevice, authorizationProvider, authorizationCode, cloudURL, cloudID string) error {
	ob := cloud.ConfigurationUpdateRequest{
		AuthorizationProvider: authorizationProvider,
		AuthorizationCode:     authorizationCode,
		URL:                   cloudURL,
		CloudID:               cloudID,
	}

	for _, l := range links.GetResourceLinks(cloud.ConfigurationResourceType) {
		err := d.UpdateResource(ctx, l, ob, nil)
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("cloud resource not found")
}

func setACLForCloud(ctx context.Context, p *core.ProvisioningClient, cloudID string, links schema.ResourceLinks) error {
	link, err := core.GetResourceLink(links, "/oic/sec/acl2")
	if err != nil {
		return err
	}

	var acls acl.Response
	err = p.GetResource(ctx, link, &acls)
	if err != nil {
		return err
	}

	for _, acl := range acls.AccessControlList {
		if acl.Subject.Subject_Device != nil {
			if acl.Subject.Subject_Device.DeviceID == cloudID {
				return nil
			}
		}
	}

	cloudACL := acl.UpdateRequest{
		AccessControlList: []acl.AccessControl{
			acl.AccessControl{
				Permission: acl.AllPermissions,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: cloudID,
					},
				},
				Resources: acl.AllResources,
			},
		},
	}

	return p.UpdateResource(ctx, link, cloudACL, nil)
}

func (c *Client) OnboardDevice(
	ctx context.Context,
	deviceID, authorizationProvider, cloudURL, authCode, cloudID string,
) error {

	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	ok, err := d.IsSecured(ctx)
	if err != nil {
		return err
	}
	if ok {
		p, err := d.Provision(ctx, links)
		if err != nil {
			return err
		}
		defer p.Close(ctx)

		err = setACLForCloud(ctx, p, cloudID, links)
		if err != nil {
			return err
		}

		return p.SetCloudResource(ctx, cloud.ConfigurationUpdateRequest{
			AuthorizationProvider: authorizationProvider,
			AuthorizationCode:     authCode,
			URL:                   cloudURL,
			CloudID:               cloudID,
		})
	}
	return setCloudResource(ctx, links, d, authorizationProvider, authCode, cloudURL, cloudID)
}
