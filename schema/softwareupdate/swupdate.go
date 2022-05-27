// Software update info.
// https://github.com/openconnectivityfoundation/core-extensions/blob/master/swagger2.0/oic.r.softwareupdate.swagger.json

package softwareupdate

const (
	ResourceType = "oic.r.softwareupdate"
	ResourceURI  = "/oc/swu"
)

type SoftwareUpdate struct {
	ResourceTypes []string     `json:"rt,omitempty"`
	Interfaces    []string     `json:"if,omitempty"`
	NewVersion    string       `json:"nv,omitempty"`
	PackageURL    string       `json:"purl,omitempty"`
	UpdateAction  UpdateAction `json:"swupdateaction,omitempty"`
	UpdateState   UpdateState  `json:"swupdatestate,omitempty"`
	UpdateResult  *int         `json:"swupdateresult,omitempty"`
	LastUpdate    string       `json:"lastupdate,omitempty"`
	Signed        Signer       `json:"signed,omitempty"`
	UpdateTime    string       `json:"updatetime,omitempty"`
}

func (sw *SoftwareUpdate) GetUpdateResult() int {
	if sw == nil || sw.UpdateResult == nil {
		return -1
	}
	return *sw.UpdateResult
}

type UpdateAction string

const (
	UpdateAction_IDLE                  UpdateAction = "idle"
	UpdateAction_CHECK_IS_AVAILABLE    UpdateAction = "isac"
	UpdateAction_DOWNLOAD_AND_VALIDATE UpdateAction = "isvv"
	UpdateAction_UPGRADE               UpdateAction = "upgrade"
)

type UpdateState string

const (
	UpdateState_IDLE                   UpdateState = "idle"
	UpdateState_NEW_SOFTWARE_AVAILABLE UpdateState = "nsa"
	UpdateState_DOWNLOADING_VALIDATING UpdateState = "svv"
	UpdateState_DOWNLOAED_VALIDATED    UpdateState = "sva"
	UpdateState_UPGRADING              UpdateState = "upgrading"
)

type Signer string

const (
	Signer_VENDOR Signer = "vendor"
)
