package cloud

// Cloud Configuration Resource
// https://github.com/openconnectivityfoundation/cloud-services/blob/master/swagger2.0/oic.r.coapcloudconf.swagger.json

const ConfigurationResourceHref = "/CoapCloudConfResURI"

const ConfigurationResourceType = "oic.r.coapcloudconf"

var ConfigurationResourceTypes = []string{ConfigurationResourceType}

// ProvisioningStatus indicates the Cloud Provisioning status of the Device.
type ProvisioningStatus string

const (
	ProvisioningStatus_UNINITIALIZED     ProvisioningStatus = "uninitialized"
	ProvisioningStatus_READY_TO_REGISTER ProvisioningStatus = "readytoregister"
	ProvisioningStatus_REGISTERING       ProvisioningStatus = "registering"
	ProvisioningStatus_REGISTERED        ProvisioningStatus = "registered"
	ProvisioningStatus_FAILED            ProvisioningStatus = "failed"
)

type Configuration struct {
	ResourceTypes         []string           `codec:"rt"`
	Interfaces            []string           `codec:"if"`
	Name                  string             `codec:"n"`
	AuthorizationProvider string             `codec:"apn"`
	CloudID               string             `codec:"sid"`
	URL                   string             `codec:"cis"`
	LastErrorCode         int                `codec:"clec"`
	ProvisioningStatus    ProvisioningStatus `codec:"cps"`
}

type ConfigurationUpdateRequest struct {
	AuthorizationProvider string `codec:"apn"`
	URL                   string `codec:"cis"`
	AuthorizationCode     string `codec:"at"`
	CloudID               string `codec:"sid"`
}
