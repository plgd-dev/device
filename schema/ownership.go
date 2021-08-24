package schema

import "fmt"

const DoxmHref = "/oic/sec/doxm"

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.doxm.swagger.json

type Doxm struct {
	ResourceOwner                 string                `json:"rowneruuid"`
	SupportedOwnerTransferMethods []OwnerTransferMethod `json:"oxms"`
	OwnerID                       string                `json:"devowneruuid"`
	DeviceID                      string                `json:"deviceuuid"`
	Owned                         bool                  `json:"owned"`
	Name                          string                `json:"n"`
	InstanceID                    string                `json:"id"`
	SupportedCredentialTypes      CredentialType        `json:"sct"`
	SelectedOwnerTransferMethod   OwnerTransferMethod   `json:"oxmsel"`
	Interfaces                    []string              `json:"if"`
	ResourceTypes                 []string              `json:"rt"`
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
