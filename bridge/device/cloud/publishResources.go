/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package cloud

import (
	"context"
	"fmt"
	"log"

	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message/codes"
)

type PublishResourcesRequest struct {
	DeviceID   string               `json:"di"`
	Links      schema.ResourceLinks `json:"links"`
	TimeToLive int                  `json:"ttl"`
}

var ErrCannotPublishResources = fmt.Errorf("cannot publish resources")

func errCannotPublishResources(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotPublishResources, err)
}

func (c *Manager) publishResources(ctx context.Context) error {
	if c.resourcesPublished {
		return nil
	}

	links := c.getLinks(schema.Endpoints{}, c.deviceID, nil, resources.PublishToCloud)
	links = patchDeviceLink(links)
	wkRd := PublishResourcesRequest{
		DeviceID:   c.deviceID.String(),
		Links:      links,
		TimeToLive: 0,
	}
	req, err := newPostRequest(ctx, c.client, ResourceDirectory, wkRd)
	if err != nil {
		return errCannotPublishResources(err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return errCannotPublishResources(err)
	}
	if resp.Code() != codes.Changed {
		return errCannotPublishResources(fmt.Errorf("unexpected status code %v", resp.Code()))
	}
	c.resourcesPublished = true
	log.Printf("resourcesPublished\n")
	return nil
}
