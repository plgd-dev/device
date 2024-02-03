/****************************************************************************
 *
 * Copyright (c) 2023 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package cloud

import (
	"github.com/plgd-dev/device/v2/schema/cloud"
)

type Configuration struct {
	ResourceTypes         []string                 `yaml:"-" json:"rt"`
	Interfaces            []string                 `yaml:"-" json:"if"`
	Name                  string                   `yaml:"-" json:"n"`
	AuthorizationProvider string                   `yaml:"authorizationProvider" json:"apn"`
	CloudID               string                   `yaml:"cloudID" json:"sid"`
	URL                   string                   `yaml:"cloudEndpoint" json:"cis"`
	LastErrorCode         int                      `yaml:"-" json:"clec"`
	ProvisioningStatus    cloud.ProvisioningStatus `yaml:"-" json:"cps"`
	AuthorizationCode     string                   `yaml:"-" json:"-"`
}
