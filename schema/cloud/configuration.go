// Package cloud implements Cloud Configuration Resource.
// https://github.com/openconnectivityfoundation/cloud-services/blob/master/swagger2.0/oic.r.coapcloudconf.swagger.json
package cloud

const (
	// ResourceType is the resource type of the Cloud Configuration Resource.
	ResourceType = "oic.r.coapcloudconf"
	// ResourceURI is the URI of the Cloud Configuration Resource.
	ResourceURI = "/CoapCloudConfResURI"
)

// ProvisioningStatus indicates the Cloud Provisioning status of the Device.
type ProvisioningStatus string

const (
	ProvisioningStatus_UNINITIALIZED     ProvisioningStatus = "uninitialized"
	ProvisioningStatus_READY_TO_REGISTER ProvisioningStatus = "readytoregister"
	ProvisioningStatus_REGISTERING       ProvisioningStatus = "registering"
	ProvisioningStatus_REGISTERED        ProvisioningStatus = "registered"
	ProvisioningStatus_FAILED            ProvisioningStatus = "failed"
)

// Configuration contains the supported fields of the Cloud Configuration Resource.
type Configuration struct {
	ResourceTypes         []string           `json:"rt"`
	Interfaces            []string           `json:"if"`
	Name                  string             `json:"n"`
	AuthorizationProvider string             `json:"apn"`
	CloudID               string             `json:"sid"`
	URL                   string             `json:"cis"`
	LastErrorCode         int                `json:"clec"`
	ProvisioningStatus    ProvisioningStatus `json:"cps"`
}

// ConfigurationUpdateRequest is used to update the Cloud Configuration Resource.
type ConfigurationUpdateRequest struct {
	AuthorizationProvider string `json:"apn"`
	URL                   string `json:"cis"`
	AuthorizationCode     string `json:"at"`
	CloudID               string `json:"sid"`
}
