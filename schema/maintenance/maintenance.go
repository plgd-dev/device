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

package maintenance

// Maintenance.
// https://github.com/openconnectivityfoundation/core-extensions/blob/master/swagger2.0/oic.wk.mnt.swagger.json

const (
	ResourceType = "oic.wk.mnt"
	ResourceURI  = "/oic/mnt"
)

type Maintenance struct {
	ResourceTypes []string `json:"rt,omitempty"`
	Interfaces    []string `json:"if,omitempty"`
	Name          string   `json:"n,omitempty"`
	FactoryReset  bool     `json:"fr,omitempty"`
	Reboot        bool     `json:"rb,omitempty"`
	LastHTTPError int      `json:"err,omitempty"`
}

type MaintenanceV1 struct {
	ResourceTypes []string `json:"rt,omitempty"`
	Interfaces    []string `json:"if,omitempty"`
	Name          string   `json:"n,omitempty"`
	FactoryReset  *bool    `json:"fr,omitempty"`
	Reboot        *bool    `json:"rb,omitempty"`
	LastHTTPError *int     `json:"err,omitempty"`
}

type MaintenanceUpdateRequest struct {
	FactoryReset bool `json:"fr,omitempty"`
	Reboot       bool `json:"rb,omitempty"`
}
