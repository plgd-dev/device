// Access Control List
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.acl2.swagger.json
package acl

import (
	"fmt"
	"strings"
)

type Response struct {
	ResourceOwner     string          `codec:"rowneruuid"`
	Interfaces        []string        `codec:"if"`
	ResourceTypes     []string        `codec:"rt"`
	Name              string          `codec:"n"`
	AccessControlList []AccessControl `codec:"aclist2"`
}

type UpdateRequest struct {
	ResourceOwner     string          `codec:"rowneruuid,omitempty"`
	AccessControlList []AccessControl `codec:"aclist2"`
}

type AccessControl struct {
	ID         int           `codec:"id,omitempty"`
	Permission Permission    `codec:"permission"`
	Resources  []Resource    `codec:"resources"`
	Subject    Subject       `codec:"subject"`
	Validity   []TimePattern `codec:"validity,omitempty"`
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
	Href          string           `codec:"href,omitempty"`
	Interfaces    []string         `codec:"if,omitempty"`
	ResourceTypes []string         `codec:"rt,omitempty"`
	Wildcard      ResourceWildcard `codec:"wc,omitempty"`
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
	DeviceID string `codec:"uuid,omitempty"`
}

type Subject_Role struct {
	Authority string `codec:"authority,omitempty"`
	Role      string `codec:"role,omitempty"`
}

type Subject_Connection struct {
	Type ConnectionType `codec:"conntype,omitempty"`
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
	Period     string `codec:"period"`
	Recurrence string `codec:"recurrence"`
}
