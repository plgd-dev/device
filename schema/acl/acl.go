// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

// Package acl implements the Access Control List resource.
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.acl2.swagger.json
package acl

import (
	"fmt"
	"strings"
)

const (
	// ResourceType is the resource type of the Access Control List resource.
	ResourceType = "oic.r.acl2"
	// ResourceURI is the URI of the Access Control List resource.
	ResourceURI = "/oic/sec/acl2"
)

// Response contains the supported fields of the Access Control List resource.
type Response struct {
	ResourceOwner     string          `json:"rowneruuid"`
	Interfaces        []string        `json:"if"`
	ResourceTypes     []string        `json:"rt"`
	Name              string          `json:"n"`
	AccessControlList []AccessControl `json:"aclist2"`
}

// UpdateRequest is used to update the Access Control List resource.
type UpdateRequest struct {
	ResourceOwner     string          `json:"rowneruuid,omitempty"`
	AccessControlList []AccessControl `json:"aclist2"`
}

// AccessControl defines permissions for one or more resources.
type AccessControl struct {
	ID         int           `json:"id,omitempty"`
	Permission Permission    `json:"permission"`
	Resources  []Resource    `json:"resources"`
	Subject    Subject       `json:"subject"`
	Tag        string        `json:"tag,omitempty"`
	Validity   []TimePattern `json:"validity,omitempty"`
}

// Permission is a bitmask encoding of CRUDN persmissions.
type Permission int

const (
	// Permission_CREATE grants permission for CREATE operations.
	Permission_CREATE Permission = 1
	// Permission_READ grants permission for RETRIEVE, OBSERVE and DISCOVER operations.
	Permission_READ Permission = 2
	// Permission_WRITE grants permission for WRITE and UPDATE operations.
	Permission_WRITE Permission = 4
	// Permission_DELETE grants permission for DELETE operations.
	Permission_DELETE Permission = 8
	// Permission_NOTIFY grants permission for NOTIFY operations.
	Permission_NOTIFY Permission = 16

	// AllPermissions is a convenience bitmask with all available permissions granted.
	AllPermissions = Permission_CREATE | Permission_READ | Permission_WRITE | Permission_DELETE | Permission_NOTIFY
)

func (p Permission) String() string {
	res := make([]string, 0, 5)
	if p.Has(Permission_CREATE) {
		res = append(res, "CREATE")
		p &^= Permission_CREATE
	}
	if p.Has(Permission_READ) {
		res = append(res, "READ")
		p &^= Permission_READ
	}
	if p.Has(Permission_WRITE) {
		res = append(res, "WRITE")
		p &^= Permission_WRITE
	}
	if p.Has(Permission_DELETE) {
		res = append(res, "DELETE")
		p &^= Permission_DELETE
	}
	if p.Has(Permission_NOTIFY) {
		res = append(res, "NOTIFY")
		p &^= Permission_NOTIFY
	}
	if p != 0 {
		res = append(res, fmt.Sprintf("unknown(%v)", int(p)))
	}
	return strings.Join(res, "|")
}

// Has returns true if the flag is set.
func (p Permission) Has(flag Permission) bool {
	return p&flag != 0
}

type Resource struct {
	Href          string           `json:"href,omitempty"`
	Interfaces    []string         `json:"if,omitempty"`
	ResourceTypes []string         `json:"rt,omitempty"`
	Wildcard      ResourceWildcard `json:"wc,omitempty"`
}

var AllResources = []Resource{
	{
		Interfaces: []string{"*"},
		Wildcard:   ResourceWildcard_NONCFG_ALL,
	},
}

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

// Subject contains anyof/oneof the subtypes
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
