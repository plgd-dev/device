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
	"hash/crc32"
	"strings"

	"github.com/plgd-dev/device/v2/bridge/resources"
	ocfCloud "github.com/plgd-dev/device/v2/pkg/ocf/cloud"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message/codes"
)

var (
	ErrCannotPublishResources   = fmt.Errorf("cannot publish resources")
	ErrCannotUnpublishResources = fmt.Errorf("cannot unpublish resources")
)

func errCannotPublishResources(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotPublishResources, err)
}

func errCannotUnpublishResources(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotUnpublishResources, err)
}

func (c *Manager) getLinksToPublish(readyResources map[string]struct{}) schema.ResourceLinks {
	if c.resourcesPublished && len(readyResources) == 0 {
		return nil
	}
	links := c.getLinks(schema.Endpoints{}, c.deviceID, nil, resources.PublishToCloud)
	patchDeviceLink(links)
	if !c.resourcesPublished {
		return links
	}
	filtered := make(schema.ResourceLinks, 0, len(readyResources))
	for _, l := range links {
		if _, ok := readyResources[l.Href]; ok {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

func (c *Manager) publishResources(ctx context.Context) error {
	readyResources := c.popReadyToPublishResources()
	links := c.getLinksToPublish(readyResources)
	if len(links) == 0 {
		return nil
	}
	wkRd := ocfCloud.PublishResourcesRequest{
		DeviceID:   c.deviceID.String(),
		Links:      links,
		TimeToLive: 0,
	}
	req, err := newPostRequest(ctx, c.client, ocfCloud.ResourceDirectory, wkRd)
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
	c.logger.Infof("resources published")
	return nil
}

func getInstanceID(href string) int64 {
	h := crc32.New(crc32.IEEETable)
	_, _ = h.Write([]byte(href))
	return int64(h.Sum32())
}

func toQuery(deviceID string, hrefs []string) string {
	var buf strings.Builder
	buf.WriteString("di=")
	buf.WriteString(deviceID)
	for _, href := range hrefs {
		buf.WriteString("&ins=")
		buf.WriteString(fmt.Sprintf("%v", getInstanceID(href)))
	}
	return buf.String()
}

func (c *Manager) unpublishResources(ctx context.Context) error {
	firstRun := true
	for {
		// take only 10 resources to unpublish in one step because of the query length limit(255 characters)
		readyResouces := c.popReadyToUnpublishResources(10)
		if len(readyResouces) == 0 {
			if firstRun {
				// to not produce log
				return nil
			}
			break
		}
		firstRun = false
		req, err := newDeleteRequest(ctx, c.client, ocfCloud.ResourceDirectory)
		if err != nil {
			return errCannotUnpublishResources(err)
		}
		query := toQuery(c.deviceID.String(), readyResouces)
		req.AddQuery(query)
		resp, err := c.client.Do(req)
		if err != nil {
			return errCannotUnpublishResources(err)
		}
		if resp.Code() != codes.Deleted {
			return errCannotUnpublishResources(fmt.Errorf("unexpected status code %v", resp.Code()))
		}
	}
	c.logger.Infof("resources unpublished")
	return nil
}
