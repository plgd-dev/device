package schema

import (
	"strings"
	"fmt"
)

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.acl2.swagger.json

type AccessControlListResponse struct {
	ResourceOwner string       `codec:"rowneruuid"`
	Interfaces    []string     `codec:"if"`
	ResourceTypes []string     `codec:"rt"`
	Name          string       `codec:"n"`
	AccessControlList []AccessControl `codec:"aclist2"`
}


type AccessControlListUpdateRequest struct {
	ResourceOwner string       `codec:"rowneruuid"`
	AccessControlList []AccessControl `codec:"aclist2"`
}


type AccessControl struct {
	ID int `codec:"id,omitempty"`
	Permission AccessControlPermission `codec:"permission"`
	Resources  []AccessControlResource  `codec:"resources"`
	Subject   AccessControlSubject `codec:"subject"`
	Validity  []AccessControlTimePattern `codec:"validity,omitempty"`
}

type AccessControlPermission int

const (
	AccessControlPermission_CREATE AccessControlPermission = 1
	AccessControlPermission_READ   AccessControlPermission = 2
	AccessControlPermission_WRITE  AccessControlPermission = 4
	AccessControlPermission_DELETE AccessControlPermission = 8
	AccessControlPermission_NOTIFY AccessControlPermission = 16
)

func (s AccessControlPermission) String() string {
	res := make([]string, 0, 5)
	if s.Has(AccessControlPermission_CREATE) {
		res = append(res, "CREATE")
		s &^= AccessControlPermission_CREATE
	}
	if s.Has(AccessControlPermission_READ) {
		res = append(res, "READ")
		s &^= AccessControlPermission_READ
	}
	if s.Has(AccessControlPermission_WRITE) {
		res = append(res, "WRITE")
		s &^= AccessControlPermission_WRITE
	}
	if s.Has(AccessControlPermission_DELETE) {
		res = append(res, "DELETE")
		s &^= AccessControlPermission_DELETE
	}
	if s.Has(AccessControlPermission_NOTIFY) {
		res = append(res, "NOTIFY")
		s &^= AccessControlPermission_NOTIFY
	}
	if s != 0 {
		res = append(res, fmt.Sprintf("unknown(%v)", int(s)))
	}
	return strings.Join(res, "|")
}

// Has returns true if the flag is set.
func (b AccessControlPermission) Has(flag AccessControlPermission) bool {
	return b&flag != 0
}

type AccessControlResource struct {
	Href string `codec:"href,omitempty"`
	Interfaces []string `codec:"if,omitempty"`
	ResourceTypes []string `codec:"rt,omitempty"`
	Wildcard AccessControlResourceWildcard `codec:"w,omitempty"`
}

type AccessControlResourceWildcard string

const(
	AccessControlResourceWildcard_NONCFG_SEC_ENDPOINT AccessControlResourceWildcard  = "+"
	AccessControlResourceWildcard_NONCFG_NONSEC_ENDPOINT AccessControlResourceWildcard  = "-"
	AccessControlResourceWildcard_NONCFG_ALL AccessControlResourceWildcard  = "*"
)

type AccessControlSubjectDevice struct {
	DeviceId string  `codec:"uuid,omitempty"`
}

type AccessControlSubjectRole struct {
	Authority string  `codec:"authority,omitempty"`
	Role string  `codec:"role,omitempty"`
}

type AccessControlSubjectConnection struct {
	Type AccessControlSubjectConnectionType  `codec:"conntype,omitempty"`
}

type AccessControlSubjectConnectionType string

const (
	AccessControlSubjectConnectionType_AUTH_CRYPT AccessControlSubjectConnectionType = "auth-crypt"
	AccessControlSubjectConnectionType_ANON_CLEAR AccessControlSubjectConnectionType = "anon-clear"
)

// anyof/oneof
type AccessControlSubject struct {
	*AccessControlSubjectDevice
	*AccessControlSubjectRole
	*AccessControlSubjectConnection
}

type AccessControlTimePattern struct {
	Period string `codec:"period"`
	Recurrence string `codec:"recurrence"`
}