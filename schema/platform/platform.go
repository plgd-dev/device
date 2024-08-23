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

// Platform info
// https://github.com/openconnectivityfoundation/core/blob/master/swagger2.0/oic.wk.p.swagger.json
package platform

import (
	"fmt"

	"github.com/plgd-dev/device/v2/internal/math"
)

const (
	ResourceType = "oic.wk.p"
	ResourceURI  = "/oic/p"
)

type Platform struct {
	Interfaces                      []string `json:"if,omitempty"`
	ResourceTypes                   []string `json:"rt,omitempty"`
	PlatformIdentifier              string   `json:"pi"`
	ManufacturerName                string   `json:"mnmn"`
	SerialNumber                    string   `json:"mnsel,omitempty"`
	ManufacturersURL                string   `json:"mnml,omitempty"`
	ManufacturersSupport            string   `json:"mnsl,omitempty"`
	ModelNumber                     string   `json:"mnmo,omitempty"`
	ManufacturersDefinedInformation string   `json:"vid,omitempty"`
	PlatformVersion                 string   `json:"mnpv,omitempty"`

	// custom properties
	Version uint32 `json:"x.org.iotivity.version,omitempty"`
}

func (p Platform) GetVersion() (uint8, uint8, uint8, uint8) {
	major := math.CastTo[uint8]((p.Version >> 24) & 0xFF)
	minor := math.CastTo[uint8]((p.Version >> 16) & 0xFF)
	patch := math.CastTo[uint8]((p.Version >> 8) & 0xFF)
	bugfix := math.CastTo[uint8](p.Version & 0xFF)
	return major, minor, patch, bugfix
}

func (p Platform) GetVersionString() string {
	major, minor, patch, bugfix := p.GetVersion()
	return fmt.Sprintf("%d.%d.%d.%d", major, minor, patch, bugfix)
}
