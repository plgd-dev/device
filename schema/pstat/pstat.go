// Device Provisioning Status
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.pstat.swagger.json
package pstat

import (
	"fmt"
	"strings"
)

const (
	ResourceType = "oic.r.pstat"
	ResourceURI  = "/oic/sec/pstat"
)

type DeviceOnboardingState struct {
	Pending                          bool             `json:"p,omitempty"`
	CurrentOrPendingOperationalState OperationalState `json:"s"`
}

type ProvisionStatusResponse struct {
	ResourceOwner             string                `json:"rowneruuid"`
	Interfaces                []string              `json:"if"`
	ResourceTypes             []string              `json:"rt"`
	CurrentOperationalMode    OperationalMode       `json:"om"`
	CurrentProvisioningMode   ProvisioningMode      `json:"cm"`
	Name                      string                `json:"n"`
	InstanceId                string                `json:"id"`
	DeviceIsOperational       bool                  `json:"isop"`
	TargetProvisioningMode    ProvisioningMode      `json:"tm"`
	SupportedOperationalModes OperationalMode       `json:"sm"`
	DeviceOnboardingState     DeviceOnboardingState `json:"dos"`
}

type ProvisionStatusUpdateRequest struct {
	ResourceOwner          string                 `json:"rowneruuid,omitempty"`
	CurrentOperationalMode OperationalMode        `json:"om,omitempty"`
	TargetProvisioningMode ProvisioningMode       `json:"tm,omitempty"`
	DeviceOnboardingState  *DeviceOnboardingState `json:"dos,omitempty"`
}

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
		return "OperationalState_SRESET"
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

func (s OperationalMode) String() string {
	res := make([]string, 0, 4)
	if s.Has(OperationalMode_SERVER_DIRECTED_UTILIZING_MULTIPLE_SERVICES) {
		res = append(res, "SERVER_DIRECTED_UTILIZING_MULTIPLE_SERVICES")
		s &^= OperationalMode_SERVER_DIRECTED_UTILIZING_MULTIPLE_SERVICES
	}
	if s.Has(OperationalMode_SERVER_DIRECTED_UTILIZING_SINGLE_SERVICE) {
		res = append(res, "SERVER_DIRECTED_UTILIZING_SINGLE_SERVICE")
		s &^= OperationalMode_SERVER_DIRECTED_UTILIZING_SINGLE_SERVICE
	}
	if s.Has(OperationalMode_CLIENT_DIRECTED) {
		res = append(res, "CLIENT_DIRECTED")
		s &^= OperationalMode_CLIENT_DIRECTED
	}
	if s != 0 {
		res = append(res, fmt.Sprintf("unknown(%v)", int(s)))
	}
	return strings.Join(res, "|")
}

// Has returns true if the flag is set.
func (b OperationalMode) Has(flag OperationalMode) bool {
	return b&flag != 0
}

type ProvisioningMode uint16

const (
	// ProvisioningMode_INIT_SOFT_VER_VALIDATION - Software version validation requested/pending(1), completed(0)
	ProvisioningMode_INIT_SOFT_VER_VALIDATION ProvisioningMode = 1 << 6
	// ProvisioningMode_INIT_SEC_SOFT_UPDATE - Secure software update requested/pending(1), completed(0)
	ProvisioningMode_INIT_SEC_SOFT_UPDATE ProvisioningMode = 1 << 7
)

func (s ProvisioningMode) String() string {
	res := make([]string, 0, 3)
	if s.Has(ProvisioningMode_INIT_SOFT_VER_VALIDATION) {
		res = append(res, "INIT_SOFT_VER_VALIDATION")
		s &^= ProvisioningMode_INIT_SOFT_VER_VALIDATION
	}
	if s.Has(ProvisioningMode_INIT_SEC_SOFT_UPDATE) {
		res = append(res, "INIT_SEC_SOFT_UPDATE")
		s &^= ProvisioningMode_INIT_SEC_SOFT_UPDATE
	}
	if s != 0 {
		res = append(res, fmt.Sprintf("unknown(%v)", int(s)))
	}
	return strings.Join(res, "|")
}

// Has returns true if the flag is set.
func (b ProvisioningMode) Has(flag ProvisioningMode) bool {
	return b&flag != 0
}
