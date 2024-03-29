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

// ResourceLink provides a link for retrieving details for its resource types:
// https://github.com/openconnectivityfoundation/core/blob/OCF-v2.0.0/schemas/oic.oic-link-schema.json
package schema

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	kitNet "github.com/plgd-dev/kit/v2/net"
	kitStrings "github.com/plgd-dev/kit/v2/strings"
)

type ResourceLink struct {
	ID                    string    `json:"id,omitempty"`
	Href                  string    `json:"href"`
	ResourceTypes         []string  `json:"rt,omitempty"`
	Interfaces            []string  `json:"if,omitempty"`
	Policy                *Policy   `json:"p,omitempty"`
	Endpoints             Endpoints `json:"eps,omitempty"`
	Anchor                string    `json:"anchor,omitempty"`
	DeviceID              string    `json:"di,omitempty"`
	InstanceID            int64     `json:"ins,omitempty"`
	Title                 string    `json:"title,omitempty"`
	SupportedContentTypes []string  `json:"type,omitempty"`
}

type Endpoints []Endpoint

type ResourceLinks []ResourceLink

// Policy is defined on the line 1822 of the Core specification:
// https://openconnectivity.org/specs/OCF_Core_Specification_v2.0.0.pdf
type Policy struct {
	BitMask    BitMask `json:"bm"`
	UDPPort    uint16  `json:"port,omitempty"`
	TCPPort    uint16  `json:"x.org.iotivity.tcp,omitempty"`
	TCPTLSPort uint16  `json:"x.org.iotivity.tls,omitempty"`

	// Secured is true if the resource is only available via an encrypted connection.
	Secured *bool `json:"sec,omitempty"`
}

// Endpoint is defined on the line 2439 and 1892, Priority on 2434 of the Core specification:
// - https://openconnectivity.org/specs/OCF_Core_Specification_v2.0.0.pdf
// - https://github.com/openconnectivityfoundation/core/blob/OCF-v2.0.0/schemas/oic.oic-link-schema.json
// When there are multiple endpoints, Priority indicates the priority among them.
// The lower the value, the higher the priority.
type Endpoint struct {
	URI      string `json:"ep"`
	Priority uint64 `json:"pri"`
}

// GetAddr parses a endpoint URI to addr.
func (ep Endpoint) GetAddr() (kitNet.Addr, error) {
	a := strings.Split(ep.URI, "://")
	if len(a) != 2 {
		return kitNet.Addr{}, fmt.Errorf("invalid address %v", ep.URI)
	}
	return kitNet.ParseString(a[0], a[1])
}

// BitMask is defined with Policy on the line 1822 of the Core specification.
type BitMask uint8

// BitMask is defined with Policy on the line 1822 of the Core specification.
const (
	Discoverable BitMask = 1 << iota
	Observable
)

// Has returns true if the flag is set.
func (b BitMask) Has(flag BitMask) bool { return b&flag != 0 }

// GetResourceHrefs resolves URIs for a resource type.
func (r ResourceLinks) GetResourceHrefs(resourceTypes ...string) []string {
	rt := make(kitStrings.Set, len(resourceTypes))
	rt.Add(resourceTypes...)
	links := make(kitStrings.Set, len(r))
	for _, r := range r {
		if rt.HasOneOf(r.ResourceTypes...) {
			links.Add(r.Href)
		}
	}
	return links.ToSlice()
}

// GetResourceLink finds a resource link with the same href.
func (r ResourceLinks) GetResourceLink(href string) (_ ResourceLink, ok bool) {
	for _, r := range r {
		if r.Href == href {
			return r, true
		}
	}
	return
}

// GetResourceLinks resolves URIs for a resource type.
func (r ResourceLinks) GetResourceLinks(resourceTypes ...string) ResourceLinks {
	rt := make(kitStrings.Set, len(resourceTypes))
	rt.Add(resourceTypes...)
	links := make([]ResourceLink, 0, len(r))
	for _, r := range r {
		if rt.HasOneOf(r.ResourceTypes...) {
			links = append(links, r)
		}
	}
	return links
}

// FilterByDeviceID filter links by device id.
func (r ResourceLinks) FilterByDeviceID(deviceID string) ResourceLinks {
	links := make([]ResourceLink, 0, len(r))
	for _, r := range r {
		if r.GetDeviceID() == deviceID {
			links = append(links, r)
		}
	}
	return links
}

// PatchEndpoint adds Endpoint information where missing.
func (r ResourceLinks) PatchEndpoint(addr kitNet.Addr, deviceEndpoints Endpoints) ResourceLinks {
	links := make(ResourceLinks, 0, len(r))
	for _, r := range r {
		links = append(links, r.PatchEndpoint(addr, deviceEndpoints))
	}
	return links
}

func (r ResourceLinks) Len() int      { return len(r) }
func (r ResourceLinks) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r ResourceLinks) Less(i, j int) bool {
	return r[i].Href < r[j].Href
}

func (r ResourceLinks) Sort() ResourceLinks {
	sort.Sort(r)
	return r
}

// GetEndpoints returns endpoints in order of priority.
func (r ResourceLink) GetEndpoints() Endpoints {
	return r.Endpoints.Sort()
}

// Sort sorts in order priority
func (r Endpoints) Sort() Endpoints {
	if len(r) == 0 {
		return r
	}
	eps := make([]Endpoint, len(r))
	copy(eps, r)
	sort.Slice(eps, func(i, j int) bool { return eps[i].Priority < eps[j].Priority })
	return eps
}

func (r Endpoints) getEndpointsWithFilter(filter func(scheme string) bool) Endpoints {
	if len(r) == 0 {
		return r
	}
	endpoints := make([]Endpoint, 0, 4)
	for _, ep := range r {
		addr, err := ep.GetAddr()
		if err != nil {
			continue
		}
		if filter(addr.GetScheme()) {
			endpoints = append(endpoints, ep)
		}
	}
	return endpoints
}

func (r ResourceLink) GetSecureEndpoints() Endpoints {
	return r.Endpoints.FilterSecureEndpoints()
}

func isSecuredScheme(scheme string) bool {
	switch scheme {
	case string(TCPSecureScheme), string(UDPSecureScheme):
		return true
	}
	return false
}

// FilterSecureEndpoints returns secure endpoints in order of priority.
func (r Endpoints) FilterSecureEndpoints() Endpoints {
	return r.getEndpointsWithFilter(isSecuredScheme)
}

func (r ResourceLink) GetUnsecureEndpoints() Endpoints {
	return r.Endpoints.FilterUnsecureEndpoints()
}

func isUnsecuredScheme(scheme string) bool {
	switch scheme {
	case string(TCPScheme), string(UDPScheme):
		return true
	}
	return false
}

// FilterUnsecureEndpoints returns unsecure endpoints in order of priority.
func (r Endpoints) FilterUnsecureEndpoints() Endpoints {
	return r.getEndpointsWithFilter(isUnsecuredScheme)
}

// HasType checks the resource type.
func (r ResourceLink) HasType(resourceType string) bool {
	for _, rt := range r.ResourceTypes {
		if rt == resourceType {
			return true
		}
	}
	return false
}

// PatchEndpoint adds Endpoint information where missing.
func (r ResourceLink) patchEndpoint(addr kitNet.Addr, deviceEndpoints Endpoints) ResourceLink {
	if len(r.Endpoints) > 0 {
		return r
	}
	if r.Policy == nil {
		return r
	}
	r.Endpoints = make([]Endpoint, 0, 4)
	if r.Policy.UDPPort != 0 {
		if r.Policy.Secured != nil && *r.Policy.Secured {
			r.Endpoints = append(r.Endpoints, udpTlsEndpoint(addr.SetPort(r.Policy.UDPPort)))
		} else {
			r.Endpoints = append(r.Endpoints, udpEndpoint(addr.SetPort(r.Policy.UDPPort)))
		}
	}
	if r.Policy.TCPPort != 0 {
		r.Endpoints = append(r.Endpoints, tcpEndpoint(addr.SetPort(r.Policy.TCPPort)))
	}
	if r.Policy.TCPTLSPort != 0 {
		r.Endpoints = append(r.Endpoints, tcpTlsEndpoint(addr.SetPort(r.Policy.TCPTLSPort)))
	}
	if len(r.Endpoints) == 0 {
		// When the device running in docker, sometimes it cannot fill endpoints for links,
		// because packet at device comes from loopback interface instead of docker_gwbridge.
		// And the loopback is not used for generating endpoints.
		// It seems like a BUG in the kernel (4.4.0-210-generic) with docker/bridge interfaces.
		if isSecuredScheme(addr.GetScheme()) {
			r.Endpoints = deviceEndpoints.FilterSecureEndpoints()
		} else {
			r.Endpoints = deviceEndpoints
		}
	}
	return r
}

// PatchEndpoint adds Endpoint information where missing.
func (r ResourceLink) PatchEndpoint(addr kitNet.Addr, deviceEndpoints Endpoints) ResourceLink {
	if len(r.Endpoints) == 0 {
		return r.patchEndpoint(addr, deviceEndpoints)
	}
	// we need to remove link-local interfaces, because we cannot determine interface
	// which need to be used in Dial
	endpoints := make([]Endpoint, 0, 8)
	for _, endpoint := range r.Endpoints {
		addrEp, err := endpoint.GetAddr()
		if err != nil {
			continue
		}
		ip, zone := kitNet.ParseIPZone(addrEp.GetHostname())
		if ip == nil {
			continue
		}
		if ip.To4() == nil && zone == "" && ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			if !strings.Contains(addr.URL(), ip.String()) {
				continue
			}
			endpoint = Endpoint{
				URI:      addr.SetScheme(addrEp.GetScheme()).SetPort(addrEp.GetPort()).URL(),
				Priority: endpoint.Priority,
			}
		}
		endpoints = append(endpoints, endpoint)
	}
	r.Endpoints = endpoints
	return r.patchEndpoint(addr, deviceEndpoints)
}

func (r Endpoints) GetAddr(scheme Scheme) (kitNet.Addr, error) {
	if len(r) == 0 {
		return kitNet.Addr{}, fmt.Errorf("no %s endpoint", scheme)
	}
	for _, ep := range r {
		u, err := ep.GetAddr()
		if err != nil {
			return kitNet.Addr{}, err
		}
		if u.GetScheme() == string(scheme) {
			return u, nil
		}
	}
	return kitNet.Addr{}, fmt.Errorf("no %s endpoint", scheme)
}

// GetTCPAddr parses and finds a TCP endpoint address.
func (r ResourceLink) GetTCPAddr() (_ kitNet.Addr, err error) {
	return r.Endpoints.GetAddr(TCPScheme)
}

// GetTCPSecureAddr parses and finds a TCP secure endpoint address.
func (r ResourceLink) GetTCPSecureAddr() (_ kitNet.Addr, err error) {
	return r.Endpoints.GetAddr(TCPSecureScheme)
}

// GetUDPAddr parses and finds a UDP endpoint address.
func (r ResourceLink) GetUDPAddr() (_ kitNet.Addr, err error) {
	return r.Endpoints.GetAddr(UDPScheme)
}

// GetUDPSecureAddr parses and finds a UDP endpoint address.
func (r ResourceLink) GetUDPSecureAddr() (_ kitNet.Addr, err error) {
	return r.Endpoints.GetAddr(UDPSecureScheme)
}

type Scheme string

const (
	TCPSecureScheme Scheme = "coaps+tcp"
	TCPScheme       Scheme = "coap+tcp"
	UDPScheme       Scheme = "coap"
	UDPSecureScheme Scheme = "coaps"
)

func udpEndpoint(addr kitNet.Addr) Endpoint {
	u := url.URL{Scheme: string(UDPScheme), Host: addr.String()}
	return Endpoint{URI: u.String()}
}

func udpTlsEndpoint(addr kitNet.Addr) Endpoint {
	u := url.URL{Scheme: string(UDPSecureScheme), Host: addr.String()}
	return Endpoint{URI: u.String()}
}

func tcpEndpoint(addr kitNet.Addr) Endpoint {
	u := url.URL{Scheme: string(TCPScheme), Host: addr.String()}
	return Endpoint{URI: u.String()}
}

func tcpTlsEndpoint(addr kitNet.Addr) Endpoint {
	u := url.URL{Scheme: string(TCPSecureScheme), Host: addr.String()}
	return Endpoint{URI: u.String()}
}

// GetDeviceID returns device id.
func (r ResourceLink) GetDeviceID() string {
	if r.DeviceID != "" {
		return r.DeviceID
	}
	return strings.TrimPrefix(r.Anchor, "ocf://")
}
