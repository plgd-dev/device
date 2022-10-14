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

// Device Owner Transfer Method
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.doxm.swagger.json
package doxm

import (
	"fmt"

	"github.com/plgd-dev/device/v2/schema/credential"
)

const (
	ResourceType = "oic.r.doxm"
	ResourceURI  = "/oic/sec/doxm"
)

type Doxm struct {
	ResourceOwner                 string                    `json:"rowneruuid"`
	SupportedOwnerTransferMethods []OwnerTransferMethod     `json:"oxms"`
	OwnerID                       string                    `json:"devowneruuid"`
	DeviceID                      string                    `json:"deviceuuid"`
	Owned                         bool                      `json:"owned"`
	Name                          string                    `json:"n"`
	InstanceID                    string                    `json:"id"`
	SupportedCredentialTypes      credential.CredentialType `json:"sct"`
	SelectedOwnerTransferMethod   OwnerTransferMethod       `json:"oxmsel"`
	Interfaces                    []string                  `json:"if"`
	ResourceTypes                 []string                  `json:"rt"`
}

type DoxmUpdate struct {
	ResourceOwner             *string              `json:"rowneruuid,omitempty"`
	OwnerID                   *string              `json:"devowneruuid,omitempty"`
	DeviceID                  *string              `json:"deviceuuid,omitempty"`
	Owned                     *bool                `json:"owned,omitempty"`
	SelectOwnerTransferMethod *OwnerTransferMethod `json:"oxmsel,omitempty"`
}

type OwnerTransferMethod int

const (
	JustWorks               = OwnerTransferMethod(0)
	SharedPin               = OwnerTransferMethod(1)
	ManufacturerCertificate = OwnerTransferMethod(2)
	Self                    = OwnerTransferMethod(4)
)

func (o OwnerTransferMethod) String() string {
	switch o {
	case JustWorks:
		return "JustWorks"
	case SharedPin:
		return "SharedPin"
	case ManufacturerCertificate:
		return "ManufacturerCertificate"
	case Self:
		return "Self"
	default:
		return fmt.Sprintf("unknown %d", o)
	}
}
