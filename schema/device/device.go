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

// Device info.
// https://github.com/openconnectivityfoundation/core/blob/master/swagger2.0/oic.wk.d.swagger.json
package device

const (
	ResourceType = "oic.wk.d"
	ResourceURI  = "/oic/d"
)

type Device struct {
	ID                    string            `json:"di"`
	ResourceTypes         []string          `json:"rt"`
	Interfaces            []string          `json:"if"`
	Name                  string            `json:"n"`
	ManufacturerName      []LocalizedString `json:"dmn"`
	ModelNumber           string            `json:"dmno"`
	ProtocolIndependentID string            `json:"piid"`
}

// LocalizedString struct.
type LocalizedString struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

// GetManufacturerName finds the manufacturer name in English.
// https://tools.ietf.org/html/rfc5646#section-2.2.1
func (d Device) GetManufacturerName() string {
	for _, n := range d.ManufacturerName {
		if n.Language == "en" {
			return n.Value
		}
	}
	return ""
}
