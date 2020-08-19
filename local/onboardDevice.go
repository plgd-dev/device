package local

import (
	"context"
	"fmt"

	"github.com/plgd-dev/sdk/schema"
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

func (c *Client) OnboardDevice(
	ctx context.Context,
	deviceID, authorizationProvider, cloudURL, authCode, cloudID string,
) error {

	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	defer d.Release(ctx)

	ok, err := d.IsSecured(ctx, links)
	if err != nil {
		return err
	}
	if ok {
		p, err := d.Provision(ctx, links)
		if err != nil {
			return err
		}
		defer p.Close(ctx)
		return p.SetCloudResource(ctx, cloud.ConfigurationUpdateRequest{
			AuthorizationProvider: authorizationProvider,
			AuthorizationCode:     authCode,
			URL:                   cloudURL,
			CloudID:               cloudID,
		})
	}
	return setCloudResource(ctx, links, d, authorizationProvider, authCode, cloudURL, cloudID)
}
