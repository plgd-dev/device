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

package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/device/v2/schema/softwareupdate"
	"github.com/plgd-dev/go-coap/v3/message"
)

func setCloudResource(ctx context.Context, links schema.ResourceLinks, d *core.Device, authorizationProvider, authorizationCode, cloudURL, cloudID string, options ...coap.OptionFunc) error {
	ob := cloud.ConfigurationUpdateRequest{
		AuthorizationProvider: authorizationProvider,
		AuthorizationCode:     authorizationCode,
		URL:                   cloudURL,
		CloudID:               cloudID,
	}

	for _, l := range links.GetResourceLinks(cloud.ResourceType) {
		return d.UpdateResource(ctx, l, ob, nil, options...)
	}

	return fmt.Errorf("cloud resource not found")
}

func setACLForCloud(ctx context.Context, p *core.ProvisioningClient, cloudID string, links schema.ResourceLinks, opts []func(message.Options) message.Options) error {
	link, err := core.GetResourceLink(links, acl.ResourceURI)
	if err != nil {
		return err
	}

	confResources := acl.AllResources
	for _, href := range links.GetResourceHrefs(softwareupdate.ResourceType) {
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

	return p.UpdateResource(ctx, link, cloudACL, nil, opts...)
}

// OnboardDevice connects device to the cloud.
// In the absence of a cached device, it is found through multicast and stored with an expiration time.
func (c *Client) OnboardDevice(
	ctx context.Context,
	deviceID, authorizationProvider, cloudURL, authCode, cloudID string,
	opts ...CommonCommandOption,
) error {
	cfg := applyCommonOptions(opts...)
	d, links, err := c.GetDevice(ctx, deviceID, cfg)
	if err != nil {
		return err
	}

	if c.useDeviceIDInQuery {
		cfg.opts = append(cfg.opts, coap.WithDeviceID(deviceID))
	}

	ok := d.IsSecured()
	if ok {
		p, err := d.Provision(ctx, links, cfg.opts...)
		if err != nil {
			return err
		}
		defer func() {
			if errC := p.Close(ctx); errC != nil {
				c.logger.Debugf("onboard device error: %w", errC)
			}
		}()

		err = setACLForCloud(ctx, p, cloudID, links, cfg.opts)
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
	return setCloudResource(ctx, links, d, authorizationProvider, authCode, cloudURL, cloudID, cfg.opts...)
}
