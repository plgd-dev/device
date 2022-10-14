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
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
)

// To support a keepalive feature, we need to filter tcp endpoints because:
// - iotivity-classic doesn't support ping over udp/dtls.
func filterTCPEndpoints(eps []schema.Endpoint) []schema.Endpoint {
	tcpDevEndpoints := make([]schema.Endpoint, 0, 4)
	for _, e := range eps {
		addr, err := e.GetAddr()
		if err != nil {
			continue
		}
		switch addr.GetScheme() {
		case string(schema.TCPScheme), string(schema.TCPSecureScheme):
			tcpDevEndpoints = append(tcpDevEndpoints, e)
		}
	}
	return tcpDevEndpoints
}

func patchResourceLinksEndpoints(links schema.ResourceLinks, disableUDPEndpoints bool) schema.ResourceLinks {
	devLink, ok := links.GetResourceLink(device.ResourceURI)
	if !ok {
		return links
	}

	tcpDevEps := devLink.GetEndpoints()
	if disableUDPEndpoints {
		tcpDevEps = filterTCPEndpoints(tcpDevEps)
	}

	patchedLinks := make(schema.ResourceLinks, 0, len(links))
	for _, l := range links {
		eps := l.GetEndpoints()
		if disableUDPEndpoints {
			eps = filterTCPEndpoints(eps)
		}
		if len(eps) == 0 {
			eps = tcpDevEps
		}
		l.Endpoints = eps
		patchedLinks = append(patchedLinks, l)
	}
	return patchedLinks
}
