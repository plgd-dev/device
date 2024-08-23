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

// Package pstat implements Device Provisioning Status resource.
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.pstat.swagger.json
package pstat

import (
	"fmt"
	"strings"
)

const (
	// ResourceType is the resource type of the Device Provisioning Status resource.
	ResourceType = "oic.r.pstat"
	// ResourceURI is the URI of the Device Provisioning Status resource.
	ResourceURI = "/oic/sec/pstat"
)

// DeviceOnboardingState contains the operation state of the device.
type DeviceOnboardingState struct {
	Pending                          bool             `json:"p,omitempty"`
	CurrentOrPendingOperationalState OperationalState `json:"s"`
}

// ProvisionStatusResponse contains the supported fields of the Device Provisioning Status resource.
type ProvisionStatusResponse struct {
	ResourceOwner             string                `json:"rowneruuid"`
	Interfaces                []string              `json:"if"`
	ResourceTypes             []string              `json:"rt"`
	CurrentOperationalMode    OperationalMode       `json:"om"`
	CurrentProvisioningMode   ProvisioningMode      `json:"cm"`
	Name                      string                `json:"n"`
	InstanceID                string                `json:"id"`
	DeviceIsOperational       bool                  `json:"isop"`
	TargetProvisioningMode    ProvisioningMode      `json:"tm"`
	SupportedOperationalModes OperationalMode       `json:"sm"`
	DeviceOnboardingState     DeviceOnboardingState `json:"dos"`
}

// ProvisionStatusUpdateRequest is used to update the Device Provisioning Status resource.
type ProvisionStatusUpdateRequest struct {
	ResourceOwner          string                 `json:"rowneruuid,omitempty"`
	CurrentOperationalMode OperationalMode        `json:"om,omitempty"`
	TargetProvisioningMode ProvisioningMode       `json:"tm,omitempty"`
	DeviceOnboardingState  *DeviceOnboardingState `json:"dos,omitempty"`
}

// OperationalState represents possible operation states of the device.
type OperationalState int

const (
	// OperationalState_RESET - Device reset state.
	OperationalState_RESET = OperationalState(0)
	// OperationalState_RFOTM - Ready for Device owner transfer method state.
	OperationalState_RFOTM = OperationalState(1)
	// OperationalState_RFPRO - Ready for Device provisioning state.
	OperationalState_RFPRO = OperationalState(2)
	// OperationalState_RFNOP - Ready for Device normal operation state.
	OperationalState_RFNOP = OperationalState(3)
	// OperationalState_SRESET - The Device is in a soft reset state."
	OperationalState_SRESET = OperationalState(4)
)

func (s OperationalState) String() string {
	switch s {
	case OperationalState_RESET:
		return "RESET"
	case OperationalState_RFOTM:
		return "RFOTM"
	case OperationalState_RFPRO:
		return "RFPRO"
	case OperationalState_RFNOP:
		return "RFNOP"
	case OperationalState_SRESET:
		return "SRESET"
	default:
		return fmt.Sprintf("unknown %v", int(s))
	}
}

type OperationalMode uint8

const (
	OperationalMode_SERVER_DIRECTED_UTILIZING_MULTIPLE_SERVICES OperationalMode = 1 << iota
	OperationalMode_SERVER_DIRECTED_UTILIZING_SINGLE_SERVICE
	OperationalMode_CLIENT_DIRECTED
)

func (m OperationalMode) String() string {
	res := make([]string, 0, 4)
	if m.Has(OperationalMode_SERVER_DIRECTED_UTILIZING_MULTIPLE_SERVICES) {
		res = append(res, "SERVER_DIRECTED_UTILIZING_MULTIPLE_SERVICES")
		m &^= OperationalMode_SERVER_DIRECTED_UTILIZING_MULTIPLE_SERVICES
	}
	if m.Has(OperationalMode_SERVER_DIRECTED_UTILIZING_SINGLE_SERVICE) {
		res = append(res, "SERVER_DIRECTED_UTILIZING_SINGLE_SERVICE")
		m &^= OperationalMode_SERVER_DIRECTED_UTILIZING_SINGLE_SERVICE
	}
	if m.Has(OperationalMode_CLIENT_DIRECTED) {
		res = append(res, "CLIENT_DIRECTED")
		m &^= OperationalMode_CLIENT_DIRECTED
	}
	if m != 0 {
		res = append(res, fmt.Sprintf("unknown(%v)", uint8(m)))
	}
	return strings.Join(res, "|")
}

// Has returns true if the flag is set.
func (m OperationalMode) Has(flag OperationalMode) bool {
	return m&flag != 0
}

type ProvisioningMode uint16

const (
	// ProvisioningMode_INIT_SOFT_VER_VALIDATION - Software version validation requested/pending(1), completed(0)
	ProvisioningMode_INIT_SOFT_VER_VALIDATION ProvisioningMode = 1 << 6
	// ProvisioningMode_INIT_SEC_SOFT_UPDATE - Secure software update requested/pending(1), completed(0)
	ProvisioningMode_INIT_SEC_SOFT_UPDATE ProvisioningMode = 1 << 7
)

func (m ProvisioningMode) String() string {
	res := make([]string, 0, 3)
	if m.Has(ProvisioningMode_INIT_SOFT_VER_VALIDATION) {
		res = append(res, "INIT_SOFT_VER_VALIDATION")
		m &^= ProvisioningMode_INIT_SOFT_VER_VALIDATION
	}
	if m.Has(ProvisioningMode_INIT_SEC_SOFT_UPDATE) {
		res = append(res, "INIT_SEC_SOFT_UPDATE")
		m &^= ProvisioningMode_INIT_SEC_SOFT_UPDATE
	}
	if m != 0 {
		res = append(res, fmt.Sprintf("unknown(%v)", uint16(m)))
	}
	return strings.Join(res, "|")
}

// Has returns true if the flag is set.
func (m ProvisioningMode) Has(flag ProvisioningMode) bool {
	return m&flag != 0
}
