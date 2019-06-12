package schema

// Cloud Configuration Resource
// https://github.com/openconnectivityfoundation/core-extensions/blob/master/swagger2.0/oic.r.coapcloudconf.swagger.json

const CloudResourceType string = "oic.r.coapcloudconf"

type CloudResponse struct {
	ResourceTypes         []string `codec:"rt"`
	Interfaces            []string `codec:"if"`
	Name                  string   `codec:"n"`
	AuthorizationProvider string   `codec:"apn"`
	CloudId               string   `codec:"sid"`
	URL                   string   `codec:"cis"`
	LastErrorCode         int      `codec:"clec"`
}

type CloudUpdateRequest struct {
	AuthorizationProvider string `codec:"apn"`
	URL                   string `codec:"cis"`
	AuthorizationCode     string `codec:"at"`
}
