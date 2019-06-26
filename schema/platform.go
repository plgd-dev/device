package schema

// Platform info
// https://github.com/openconnectivityfoundation/core/blob/master/swagger2.0/oic.wk.p.swagger.json
type Platform struct {
	Interfaces                      []string `codec:"if,omitempty"`
	ResourceTypes                   []string `codec:"rt,omitempty"`
	PlatformIdentifier              string   `codec:"pi"`
	ManufacturerName                string   `codec:"mnmn"`
	SerialNumber                    string   `codec:"mnsel,omitempty"`
	ManufacturersURL                string   `codec:"mnml,omitempty"`
	ManufacturersSupport            string   `codec:"mnsl,omitempty"`
	ModelNumber                     string   `codec:"mnmo,omitempty"`
	ManufacturersDefinedInformation string   `codec:"vid,omitempty"`
	PlatformVersion                 string   `codec:"mnpv,omitempty"`
}
