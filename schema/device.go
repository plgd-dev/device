package schema

const DeviceResourceType = "oic.wk.d"

// Device info.
// https://github.com/openconnectivityfoundation/core/blob/master/swagger2.0/oic.wk.d.swagger.json
type Device struct {
	ID               string            `json:"di"`
	ResourceTypes    []string          `json:"rt"`
	Interfaces       []string          `json:"if"`
	Name             string            `json:"n"`
	ManufacturerName []LocalizedString `json:"dmn"`
	ModelNumber      string            `json:"dmno"`
}

// LocalizedString struct.
type LocalizedString struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

// GetManufacturerName finds the manufacturer name in English.
// https://tools.ietf.org/html/rfc5646#section-2.2.1
func (d Device) GetManufacturerName() string {
	for _, n := range d.ManufacturerName {
		if n.Language == "en" {
			return n.Value
		}
	}
	return ""
}
