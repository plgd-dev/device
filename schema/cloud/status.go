package cloud

import "github.com/plgd-dev/device/schema/interfaces"

const (
	StatusResourceType = "x.cloud.device.status"
	StatusResourceURI  = "/oic/cloud/s"
)

var (
	StatusInterfaces    = []string{interfaces.OC_IF_BASELINE}
	StatusResourceTypes = []string{StatusResourceType}
)

// Status is resource published by OCF Cloud.
// - signup: resource published
// - signin: content changed -> online true
// - signout/close connection: content changed -> online false
type Status struct {
	ResourceTypes []string `json:"rt"`
	Interfaces    []string `json:"if"`
	Online        bool     `json:"online"`
}
