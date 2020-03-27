package schema

// Platform info
// https://github.com/openconnectivityfoundation/core/blob/master/swagger2.0/oic.wk.p.swagger.json
type Platform struct {
	Interfaces                      []string `json:"if,omitempty"`
	ResourceTypes                   []string `json:"rt,omitempty"`
	PlatformIdentifier              string   `json:"pi"`
	ManufacturerName                string   `json:"mnmn"`
	SerialNumber                    string   `json:"mnsel,omitempty"`
	ManufacturersURL                string   `json:"mnml,omitempty"`
	ManufacturersSupport            string   `json:"mnsl,omitempty"`
	ModelNumber                     string   `json:"mnmo,omitempty"`
	ManufacturersDefinedInformation string   `json:"vid,omitempty"`
	PlatformVersion                 string   `json:"mnpv,omitempty"`
}
