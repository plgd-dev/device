// Discoverable Resources
// https://github.com/openconnectivityfoundation/core/blob/master/swagger2.0/oic.wk.res.swagger.json
package resources

import (
	"sort"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

const (
	ResourceType = "oic.wk.res"
	ResourceURI  = "/oic/res"
)

type BatchResourceDiscovery []BatchRepresentation

func (v BatchResourceDiscovery) Len() int {
	return len(v)
}

func (v BatchResourceDiscovery) Less(i, j int) bool {
	return v[i].HrefRaw < v[j].HrefRaw
}

func (v BatchResourceDiscovery) Swap(i, j int) {
	tmp := v[i]
	v[i] = v[j]
	v[j] = tmp
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
