package local

import ocfSchema "github.com/go-ocf/sdk/schema"

// To support a keepalive feature, we need to filter tcp endpoints because:
// - iotivity-classic doesn't support ping over udp/dtls.
func filterTCPEndpoints(eps []ocfSchema.Endpoint) []ocfSchema.Endpoint {
	tcpDevEndpoints := make([]ocfSchema.Endpoint, 0, 4)
	for _, e := range eps {
		addr, err := e.GetAddr()
		if err != nil {
			continue
		}
		switch addr.GetScheme() {
		case string(ocfSchema.TCPScheme), string(ocfSchema.TCPSecureScheme):
			tcpDevEndpoints = append(tcpDevEndpoints, e)
		}
	}
	return tcpDevEndpoints
}

func (c *Client) patchResourceLinks(links ocfSchema.ResourceLinks) ocfSchema.ResourceLinks {
	devLink, ok := links.GetResourceLink("/oic/d")
	if !ok {
		return links
	}

	tcpDevEps := devLink.GetEndpoints()
	if c.disableUDPEndpoints {
		tcpDevEps = filterTCPEndpoints(tcpDevEps)
	}

	patchedLinks := make(ocfSchema.ResourceLinks, 0, len(links))
	for _, l := range links {
		eps := l.GetEndpoints()
		if c.disableUDPEndpoints {
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
