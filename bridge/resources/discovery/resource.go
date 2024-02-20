/****************************************************************************
 *
 * Copyright (c) 2023 plgd.dev s.r.o.
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

package discovery

import (
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	plgdResources "github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

type GetLinksHandler func(*net.Request) schema.ResourceLinks

type Resource struct {
	*resources.Resource
	getLinks GetLinksHandler
}

func New(uri string, getLinks GetLinksHandler) *Resource {
	d := &Resource{
		getLinks: getLinks,
	}
	d.Resource = resources.NewResource(uri, d.Get, nil, []string{plgdResources.ResourceType}, []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_R})
	return d
}

func PatchLinks(links schema.ResourceLinks, deviceID string) schema.ResourceLinks {
	for _, l := range links {
		l.Anchor = "ocf://" + deviceID
	}
	return links
}

func (d *Resource) Get(request *net.Request) (*pool.Message, error) {
	links := PatchLinks(d.getLinks(request), request.DeviceID().String())
	return resources.CreateResponseContent(request.Context(), links, codes.Content)
}
