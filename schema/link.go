package schema

import (
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"

	kitNet "github.com/go-ocf/kit/net"
	kitStrings "github.com/go-ocf/kit/strings"
)

// ResourceLink provides a link for retrieving details for its resource types:
// https://github.com/openconnectivityfoundation/core/blob/OCF-v2.0.0/schemas/oic.oic-link-schema.json
type ResourceLink struct {
	ID                    string     `codec:"id"`
	Href                  string     `codec:"href"`
	ResourceTypes         []string   `codec:"rt"`
	Interfaces            []string   `codec:"if"`
	Policy                Policy     `codec:"p"`
	Endpoints             []Endpoint `codec:"eps"`
	Anchor                string     `codec:"anchor"`
	DeviceID              string     `codec:"di"`
	InstanceID            int64      `codec:"ins"`
	Title                 string     `codec:"title"`
	SupportedContentTypes []string   `codec:"type"`
}

type ResourceLinks []ResourceLink

// Policy is defined on the line 1822 of the Core specification:
// https://openconnectivity.org/specs/OCF_Core_Specification_v2.0.0.pdf
type Policy struct {
	BitMask    BitMask `codec:"bm"`
	UDPPort    uint16  `codec:"port"`
	TCPPort    uint16  `codec:"x.org.iotivity.tcp"`
	TCPTLSPort uint16  `codec:"x.org.iotivity.tls"`

	// Secured is true if the resource is only available via an encrypted connection.
	Secured bool `codec:"sec"`
}

// Endpoint is defined on the line 2439 and 1892, Priority on 2434 of the Core specification:
// - https://openconnectivity.org/specs/OCF_Core_Specification_v2.0.0.pdf
// - https://github.com/openconnectivityfoundation/core/blob/OCF-v2.0.0/schemas/oic.oic-link-schema.json
// When there are multiple endpoints, Priority indicates the priority among them.
// The lower the value, the higher the priority.
type Endpoint struct {
	URI      string `codec:"ep"`
	Priority uint64 `codec:"pri"`
}

// GetAddr parses a endpoint URI to addr.
func (ep Endpoint) GetAddr() (kitNet.Addr, error) {
	u, err := url.ParseRequestURI(ep.URI)
	if err != nil {
		return kitNet.Addr{}, err
	}
	return kitNet.ParseURL(u)
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
func (d ResourceLinks) GetResourceHrefs(resourceTypes ...string) []string {
	rt := make(kitStrings.Set, len(resourceTypes))
	rt.Add(resourceTypes...)
	links := make(kitStrings.Set, len(d))
	for _, r := range d {
		if rt.HasOneOf(r.ResourceTypes...) {
			links.Add(r.Href)
		}
	}
	return links.ToSlice()
}

// GetResourceLink finds a resource link with the same href.
func (d ResourceLinks) GetResourceLink(href string) (_ ResourceLink, ok bool) {
	for _, r := range d {
		if r.Href == href {
			return r, true
		}
	}
	return
}

// GetResourceLinks resolves URIs for a resource type.
func (d ResourceLinks) GetResourceLinks(resourceTypes ...string) ResourceLinks {
	rt := make(kitStrings.Set, len(resourceTypes))
	rt.Add(resourceTypes...)
	links := make([]ResourceLink, 0, len(d))
	for _, r := range d {
		if rt.HasOneOf(r.ResourceTypes...) {
			links = append(links, r)
		}
	}
	return links
}

// PatchEndpoint adds Endpoint information where missing.
func (d ResourceLinks) PatchEndpoint(addr kitNet.Addr) ResourceLinks {
	links := make(ResourceLinks, 0, len(d))
	for _, r := range d {
		links = append(links, r.PatchEndpoint(addr))
	}
	return links
}

// GetEndpoints returns endpoints in order of priority.
func (r ResourceLink) GetEndpoints() []Endpoint {
	eps := make([]Endpoint, len(r.Endpoints))
	copy(eps, r.Endpoints)
	sort.Slice(eps, func(i, j int) bool { return eps[i].Priority < eps[j].Priority })
	return eps
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
func (r ResourceLink) patchEndpoint(addr kitNet.Addr) ResourceLink {
	if len(r.Endpoints) > 0 {
		return r
	}
	r.Endpoints = make([]Endpoint, 0, 4)
	if r.Policy.UDPPort != 0 {
		if r.Policy.Secured {
			r.Endpoints = append(r.Endpoints, udpTlsEndpoint(addr.SetPort(r.Policy.UDPPort)))
		} else {
			r.Endpoints = append(r.Endpoints, udpEndpoint(addr.SetPort(r.Policy.UDPPort)))
		}
	} else {
		if !r.Policy.Secured {
			r.Endpoints = append(r.Endpoints, udpEndpoint(addr))
		}
	}
	if r.Policy.TCPPort != 0 {
		r.Endpoints = append(r.Endpoints, tcpEndpoint(addr.SetPort(r.Policy.TCPPort)))
	}
	if r.Policy.TCPTLSPort != 0 {
		r.Endpoints = append(r.Endpoints, tcpTlsEndpoint(addr.SetPort(r.Policy.TCPTLSPort)))
	}
	return r
}

// PatchEndpoint adds Endpoint information where missing.
func (r ResourceLink) PatchEndpoint(addr kitNet.Addr) ResourceLink {
	if len(r.Endpoints) > 0 {
		// we need to remove link-local interfaces, because we cannot determine interface
		// which need to be used in Dial
		endpoints := make([]Endpoint, 0, 8)
		for _, endpoint := range r.Endpoints {
			url, err := url.Parse(endpoint.URI)
			if err != nil {
				continue
			}
			ip := net.ParseIP(url.Hostname())
			if ip == nil {
				continue
			}
			if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				continue
			}
			endpoints = append(endpoints, endpoint)
		}
		r.Endpoints = endpoints
		return r.patchEndpoint(addr)
	}
	return r.patchEndpoint(addr)
}

func (r ResourceLink) getEndpoint(scheme string) (_ kitNet.Addr, err error) {
	var u *url.URL
	for _, ep := range r.Endpoints {
		u, err = url.ParseRequestURI(ep.URI)
		if err != nil {
			return
		}
		if u.Scheme == scheme {
			return kitNet.ParseURL(u)
		}
	}
	err = fmt.Errorf("no %s endpoint for %v", scheme, r)
	return
}

// GetTCPAddr parses and finds a TCP endpoint address.
func (r ResourceLink) GetTCPAddr() (_ kitNet.Addr, err error) {
	return r.getEndpoint(TCPScheme)
}

// GetTCPSecureAddr parses and finds a TCP secure endpoint address.
func (r ResourceLink) GetTCPSecureAddr() (_ kitNet.Addr, err error) {
	return r.getEndpoint(TCPSecureScheme)
}

// GetUDPAddr parses and finds a UDP endpoint address.
func (r ResourceLink) GetUDPAddr() (_ kitNet.Addr, err error) {
	return r.getEndpoint(UDPScheme)
}

// GetUDPSecureAddr parses and finds a UDP endpoint address.
func (r ResourceLink) GetUDPSecureAddr() (_ kitNet.Addr, err error) {
	return r.getEndpoint(UDPSecureScheme)
}

const (
	TCPSecureScheme = "coaps+tcp"
	TCPScheme       = "coap+tcp"
	UDPScheme       = "coap"
	UDPSecureScheme = "coaps"
)

func udpEndpoint(addr kitNet.Addr) Endpoint {
	u := url.URL{Scheme: UDPScheme, Host: addr.String()}
	return Endpoint{URI: u.String()}
}

func udpTlsEndpoint(addr kitNet.Addr) Endpoint {
	u := url.URL{Scheme: UDPSecureScheme, Host: addr.String()}
	return Endpoint{URI: u.String()}
}

func tcpEndpoint(addr kitNet.Addr) Endpoint {
	u := url.URL{Scheme: TCPScheme, Host: addr.String()}
	return Endpoint{URI: u.String()}
}

func tcpTlsEndpoint(addr kitNet.Addr) Endpoint {
	u := url.URL{Scheme: TCPSecureScheme, Host: addr.String()}
	return Endpoint{URI: u.String()}
}

// GetDeviceID returns device id.
func (r ResourceLink) GetDeviceID() string {
	if r.DeviceID != "" {
		return r.DeviceID
	}
	return strings.TrimPrefix(r.Anchor, "ocf://")
}
