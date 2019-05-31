package schema

import "fmt"

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.doxm.swagger.json

type Doxm struct {
	ResourceOwner                 string   `codec:"rowneruuid"`
	SupportedOwnerTransferMethods []OwnerTransferMethod    `codec:"oxms"`
	DeviceOwner                   string   `codec:"devowneruuid"`
	DeviceId                      string   `codec:"deviceuuid"`
	Owned                         bool     `codec:"owned"`
	Name                          string   `codec:"n"`
	InstanceId                    string   `codec:"id"`
	SupportedCredentialTypes      int      `codec:"sct"`
	SelectedOwnerTransferMethod   OwnerTransferMethod      `codec:"oxmsel"`
	Interfaces                    []string `codec:"if"`
	ResourceTypes                 []string `codec:"rt"`
}

type DoxmUpdate struct {
	ResourceOwner             string `codec:"rowneruuid,omitempty"`
	DeviceOwner               string `codec:"devowneruuid,omitempty"`
	DeviceId                  string `codec:"deviceuuid,omitempty"`
	Owned                     bool   `codec:"owned,omitempty"`
	SelectOwnerTransferMethod OwnerTransferMethod `codec:"oxmsel,omitempty"`
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