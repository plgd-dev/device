package maintenance

// Maintenance.
// https://github.com/openconnectivityfoundation/core-extensions/blob/master/swagger2.0/oic.wk.mnt.swagger.json
const MaintenanceResourceType = "oic.wk.mnt"

type Maintenance struct {
	ResourceTypes []string `codec:"rt"`
	Interfaces    []string `codec:"if"`
	Name          string   `codec:"n"`
	FactoryReset  bool     `codec:"fr"`
	Reboot        bool     `codec:"rb"`
	LastHTTPError int      `codec:"err"`
}

type MaintenanceUpdateRequest struct {
	FactoryReset bool `codec:"fr,omitempty"`
	Reboot       bool `codec:"rb,omitempty"`
}
