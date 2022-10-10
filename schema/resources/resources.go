// Discoverable Resources
// https://github.com/openconnectivityfoundation/core/blob/master/swagger2.0/oic.wk.res.swagger.json
package resources

import (
	"sort"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/plgd-dev/device/v2/schema"
)

const (
	ResourceType = "oic.wk.res"
	ResourceURI  = "/oic/res"
)

type BaselineResourceDiscovery []BaselineRepresentation

type BaselineRepresentation struct {
	Interfaces    []string             `json:"if,omitempty"`
	ResourceTypes []string             `json:"rt,omitempty"`
	Links         schema.ResourceLinks `json:"links"`
}

type BatchResourceDiscovery []BatchRepresentation

func (v BatchResourceDiscovery) Len() int {
	return len(v)
}

func (v BatchResourceDiscovery) Less(i, j int) bool {
	return v[i].HrefRaw < v[j].HrefRaw
}

func (v BatchResourceDiscovery) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v BatchResourceDiscovery) Sort() {
	sort.Sort(v)
}

type BatchRepresentation struct {
	HrefRaw string          `json:"href"`
	Content cbor.RawMessage `json:"rep"`
}

func (v BatchRepresentation) DeviceID() string {
	p := strings.SplitN(strings.TrimPrefix(v.HrefRaw, "ocf://"), "/", 2)
	if len(p) != 2 {
		return ""
	}
	return p[0]
}

func (v BatchRepresentation) Href() string {
	p := strings.SplitN(strings.TrimPrefix(v.HrefRaw, "ocf://"), "/", 2)
	if len(p) != 2 {
		return ""
	}
	return "/" + p[1]
}
