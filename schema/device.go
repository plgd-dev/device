package schema

// Device info.
// https://github.com/openconnectivityfoundation/core/blob/master/schemas/oic.wk.d-schema.json
type Device struct {
	ID               string             `codec:"di"`
	Types            []string           `codec:"rt"`
	Name             string             `codec:"n"`
	ManufacturerName []ManufacturerName `codec:"dmn"`
	ModelNumber      string             `codec:"dmno"`
	SerialNumber     string             `codec:"piid"`
}

// ManufacturerName struct.
type ManufacturerName struct {
	Language string `codec:"language"`
	Value    string `codec:"value"`
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
