// Access Control List
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.acl2.swagger.json
package acl

import (
	"fmt"
	"strings"
)

type Response struct {
	ResourceOwner     string          `json:"rowneruuid"`
	Interfaces        []string        `json:"if"`
	ResourceTypes     []string        `json:"rt"`
	Name              string          `json:"n"`
	AccessControlList []AccessControl `json:"aclist2"`
}

type UpdateRequest struct {
	ResourceOwner     string          `json:"rowneruuid,omitempty"`
	AccessControlList []AccessControl `json:"aclist2"`
}

type AccessControl struct {
	ID         int           `json:"id,omitempty"`
	Permission Permission    `json:"permission"`
	Resources  []Resource    `json:"resources"`
	Subject    Subject       `json:"subject"`
	Validity   []TimePattern `json:"validity,omitempty"`
}

type Permission int

const (
	Permission_CREATE Permission = 1
	Permission_READ   Permission = 2
	Permission_WRITE  Permission = 4
	Permission_DELETE Permission = 8
	Permission_NOTIFY Permission = 16

	AllPermissions = Permission_CREATE | Permission_READ | Permission_WRITE | Permission_DELETE | Permission_NOTIFY
)

func (s Permission) String() string {
	res := make([]string, 0, 5)
	if s.Has(Permission_CREATE) {
		res = append(res, "CREATE")
		s &^= Permission_CREATE
	}
	if s.Has(Permission_READ) {
		res = append(res, "READ")
		s &^= Permission_READ
	}
	if s.Has(Permission_WRITE) {
		res = append(res, "WRITE")
		s &^= Permission_WRITE
	}
	if s.Has(Permission_DELETE) {
		res = append(res, "DELETE")
		s &^= Permission_DELETE
	}
	if s.Has(Permission_NOTIFY) {
		res = append(res, "NOTIFY")
		s &^= Permission_NOTIFY
	}
	if s != 0 {
		res = append(res, fmt.Sprintf("unknown(%v)", int(s)))
	}
	return strings.Join(res, "|")
}

// Has returns true if the flag is set.
func (b Permission) Has(flag Permission) bool {
	return b&flag != 0
}

type Resource struct {
	Href          string           `json:"href,omitempty"`
	Interfaces    []string         `json:"if,omitempty"`
	ResourceTypes []string         `json:"rt,omitempty"`
	Wildcard      ResourceWildcard `json:"wc,omitempty"`
}

var AllResources = []Resource{Resource{
	Interfaces: []string{"*"},
	Wildcard:   ResourceWildcard_NONCFG_ALL,
}}

type ResourceWildcard string

const (
	ResourceWildcard_NONCFG_SEC_ENDPOINT    ResourceWildcard = "+"
	ResourceWildcard_NONCFG_NONSEC_ENDPOINT ResourceWildcard = "-"
	ResourceWildcard_NONCFG_ALL             ResourceWildcard = "*"
)

type Subject_Device struct {
	DeviceID string `json:"uuid,omitempty"`
}

type Subject_Role struct {
	Authority string `json:"authority,omitempty"`
	Role      string `json:"role,omitempty"`
}

type Subject_Connection struct {
	Type ConnectionType `json:"conntype,omitempty"`
}

type ConnectionType string

const (
	// authenticated encrypted connection
	ConnectionType_AUTH_CRYPT ConnectionType = "auth-crypt"
	// anonymous clear-text connection
	ConnectionType_ANON_CLEAR ConnectionType = "anon-clear"
)

// anyof/oneof
type Subject struct {
	*Subject_Device
	*Subject_Role
	*Subject_Connection
}

var TLSConnection = Subject{
	Subject_Connection: &Subject_Connection{
		Type: ConnectionType_AUTH_CRYPT,
	},
}

type TimePattern struct {
	Period     string `json:"period"`
	Recurrence string `json:"recurrence"`
}
