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
