package schema

import "fmt"

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.doxm.swagger.json

type Doxm struct {
	ResourceOwner                 string   `codec:"rowneruuid"`
	SupportedOwnerTransferMethods []int    `codec:"oxms"`
	DeviceOwner                   string   `codec:"devowneruuid"`
	DeviceId                      string   `codec:"deviceuuid"`
	Owned                         bool     `codec:"owned"`
	Name                          string   `codec:"n"`
	InstanceId                    string   `codec:"id"`
	SupportedCredentialTypes      int      `codec:"sct"`
	SelectedOwnerTransferMethod   int      `codec:"oxmsel"`
	Interfaces                    []string `codec:"if"`
	ResourceTypes                 []string `codec:"rt"`
}

type DoxmSelectOwnerTransferMethod struct {
	SelectOwnerTransferMethod int `codec:"oxmsel"`
}

type DoxmUpdate struct {
	ResourceOwner             string `codec:"rowneruuid"`
	DeviceOwner               string `codec:"devowneruuid"`
	DeviceId                  string `codec:"deviceuuid"`
	Owned                     bool   `codec:"owned"`
	SelectOwnerTransferMethod int    `codec:"oxmsel"`
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

func (d Doxm) GetSupportedOwnerTransferMethods() []OwnerTransferMethod {
	r := make([]OwnerTransferMethod, 0, len(d.SupportedOwnerTransferMethods))
	for _, m := range d.SupportedOwnerTransferMethods {
		r = append(r, OwnerTransferMethod(m))
	}
	return r
}
