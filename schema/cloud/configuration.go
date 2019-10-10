package cloud

// Cloud Configuration Resource
// https://github.com/openconnectivityfoundation/core-extensions/blob/master/swagger2.0/oic.r.coapcloudconf.swagger.json

const ConfigurationResourceType string = "oic.r.coapcloudconf"

type Configuration struct {
	ResourceTypes         []string `codec:"rt"`
	Interfaces            []string `codec:"if"`
	Name                  string   `codec:"n"`
	AuthorizationProvider string   `codec:"apn"`
	CloudID               string   `codec:"sid"`
	URL                   string   `codec:"cis"`
	LastErrorCode         int      `codec:"clec"`
}

type ConfigurationUpdateRequest struct {
	AuthorizationProvider string `codec:"apn"`
	URL                   string `codec:"cis"`
	AuthorizationCode     string `codec:"at"`
	CloudID               string `codec:"sid"`
}
