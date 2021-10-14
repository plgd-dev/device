package maintenance

// Maintenance.
// https://github.com/openconnectivityfoundation/core-extensions/blob/master/swagger2.0/oic.wk.mnt.swagger.json

const (
	ResourceType = "oic.wk.mnt"
	ResourceURI  = "/oic/mnt"
)

type Maintenance struct {
	ResourceTypes []string `json:"rt"`
	Interfaces    []string `json:"if"`
	Name          string   `json:"n"`
	FactoryReset  bool     `json:"fr"`
	Reboot        bool     `json:"rb"`
	LastHTTPError int      `json:"err"`
}

type MaintenanceUpdateRequest struct {
	FactoryReset bool `json:"fr,omitempty"`
	Reboot       bool `json:"rb,omitempty"`
}
