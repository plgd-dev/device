package cloud

const StatusResourceType string = "x.cloud.device.status"

var StatusInterfaces = []string{"oic.if.baseline"}
var StatusResourceTypes = []string{StatusResourceType}

const StatusHref = "/oic/cloud/s"

// Status is resource published by OCF Cloud.
// - signup: resource published
// - signin: content changed -> online true
// - signout/close connection: content changed -> online false
type Status struct {
	ResourceTypes []string `json:"rt"`
	Interfaces    []string `json:"if"`
	Online        bool     `json:"online"`
}
