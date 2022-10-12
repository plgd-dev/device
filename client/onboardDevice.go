package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/softwareupdate"
)

func setCloudResource(ctx context.Context, links schema.ResourceLinks, d *core.Device, authorizationProvider, authorizationCode, cloudURL, cloudID string) error {
	ob := cloud.ConfigurationUpdateRequest{
		AuthorizationProvider: authorizationProvider,
		AuthorizationCode:     authorizationCode,
		URL:                   cloudURL,
		CloudID:               cloudID,
	}

	for _, l := range links.GetResourceLinks(cloud.ResourceType) {
		return d.UpdateResource(ctx, l, ob, nil)
	}

	return fmt.Errorf("cloud resource not found")
}

func setACLForCloud(ctx context.Context, p *core.ProvisioningClient, cloudID string, links schema.ResourceLinks) error {
	link, err := core.GetResourceLink(links, acl.ResourceURI)
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
	confResources := acl.AllResources
	for _, href := range links.GetResourceHrefs(softwareupdate.ResourceType) {
		confResources = append(confResources, acl.Resource{
			Href:       href,
			Interfaces: []string{"*"},
		})
	}

	cloudACL := acl.UpdateRequest{
		AccessControlList: []acl.AccessControl{
			{
				Permission: acl.AllPermissions,
				Subject: acl.Subject{
					Subject_Device: &acl.Subject_Device{
						DeviceID: cloudID,
					},
				},
				Resources: confResources,
			},
		},
	}

	return p.UpdateResource(ctx, link, cloudACL, nil)
}

func (c *Client) OnboardDevice(
	ctx context.Context,
	deviceID, authorizationProvider, cloudURL, authCode, cloudID string,
	opts ...CommonCommandOption,
) error {
	cfg := applyCommonOptions(opts...)
	d, links, err := c.GetDeviceByMulticast(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}

	ok := d.IsSecured()
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
