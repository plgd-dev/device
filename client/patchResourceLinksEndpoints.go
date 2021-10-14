package client

import (
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/device"
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
