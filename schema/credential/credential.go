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

// Credential
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.cred.swagger.json
package credential

import (
	"fmt"
	"strings"

	"github.com/plgd-dev/device/v2/schema/csr"
)

const (
	ResourceType = "oic.r.cred"
	ResourceURI  = "/oic/sec/cred"
)

type Credential struct {
	ID                      int                       `json:"credid,omitempty" yaml:"id,omitempty"`
	Type                    CredentialType            `json:"credtype" yaml:"type"`
	Subject                 string                    `json:"subjectuuid" yaml:"subject"`
	Usage                   CredentialUsage           `json:"credusage,omitempty" yaml:"usage,omitempty"`
	SupportedRefreshMethods []CredentialRefreshMethod `json:"crms,omitempty" yaml:"supportedRefreshMethods,omitempty"`
	OptionalData            *CredentialOptionalData   `json:"optionaldata,omitempty" yaml:"optionalData,omitempty"`
	Period                  string                    `json:"period,omitempty" yaml:"period,omitempty"`
	PrivateData             *CredentialPrivateData    `json:"privatedata,omitempty" yaml:"privateData,omitempty"`
	PublicData              *CredentialPublicData     `json:"publicdata,omitempty" yaml:"publicData,omitempty"`
	RoleID                  *CredentialRoleID         `json:"roleid,omitempty" yaml:"roleID,omitempty"`
	Tag                     string                    `json:"tag,omitempty" yaml:"tag,omitempty"`
}

type CredentialType uint16

const (
	CredentialType_EMPTY                               CredentialType = 0
	CredentialType_SYMMETRIC_PAIR_WISE                 CredentialType = 1
	CredentialType_SYMMETRIC_GROUP                     CredentialType = 2
	CredentialType_ASYMMETRIC_SIGNING                  CredentialType = 4
	CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE CredentialType = 8
	CredentialType_PIN_OR_PASSWORD                     CredentialType = 16
	CredentialType_ASYMMETRIC_ENCRYPTION_KEY           CredentialType = 32
)

func (c CredentialType) String() string {
	if c == CredentialType_EMPTY {
		return "EMPTY"
	}
	res := make([]string, 0, 7)
	if c.Has(CredentialType_SYMMETRIC_PAIR_WISE) {
		res = append(res, "SYMMETRIC_PAIR_WISE")
		c &^= CredentialType_SYMMETRIC_PAIR_WISE
	}
	if c.Has(CredentialType_SYMMETRIC_GROUP) {
		res = append(res, "SYMMETRIC_GROUP")
		c &^= CredentialType_SYMMETRIC_GROUP
	}
	if c.Has(CredentialType_ASYMMETRIC_SIGNING) {
		res = append(res, "ASYMMETRIC_SIGNING")
		c &^= CredentialType_ASYMMETRIC_SIGNING
	}
	if c.Has(CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE) {
		res = append(res, "ASYMMETRIC_SIGNING_WITH_CERTIFICATE")
		c &^= CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE
	}
	if c.Has(CredentialType_PIN_OR_PASSWORD) {
		res = append(res, "PIN_OR_PASSWORD")
		c &^= CredentialType_PIN_OR_PASSWORD
	}
	if c.Has(CredentialType_ASYMMETRIC_ENCRYPTION_KEY) {
		res = append(res, "ASYMMETRIC_ENCRYPTION_KEY")
		c &^= CredentialType_ASYMMETRIC_ENCRYPTION_KEY
	}
	if c != 0 {
		res = append(res, fmt.Sprintf("unknown(%v)", uint16(c)))
	}
	return strings.Join(res, "|")
}

// Has returns true if the flag is set.
func (c CredentialType) Has(flag CredentialType) bool {
	return c&flag != 0
}

type CredentialUsage string

const (
	CredentialUsage_TRUST_CA     CredentialUsage = "oic.sec.cred.trustca"    //nolint:gosec
	CredentialUsage_CERT         CredentialUsage = "oic.sec.cred.cert"       //nolint:gosec
	CredentialUsage_ROLE_CERT    CredentialUsage = "oic.sec.cred.rolecert"   //nolint:gosec
	CredentialUsage_MFG_TRUST_CA CredentialUsage = "oic.sec.cred.mfgtrustca" //nolint:gosec
	CredentialUsage_MFG_CERT     CredentialUsage = "oic.sec.cred.mfgcert"    //nolint:gosec
)

type CredentialRefreshMethod string

const (
	CredentialRefreshMethod_PROVISION_SERVICE                     CredentialRefreshMethod = "oic.sec.crm.pro"  //nolint:gosec
	CredentialRefreshMethod_KEY_AGREEMENT_PROTOCOL_AND_RANDOM_PIN CredentialRefreshMethod = "oic.sec.crm.psk"  //nolint:gosec
	CredentialRefreshMethod_KEY_AGREEMENT_PROTOCOL                CredentialRefreshMethod = "oic.sec.crm.rdp"  //nolint:gosec
	CredentialRefreshMethod_KEY_DISTRIBUTION_SERVICE              CredentialRefreshMethod = "oic.sec.crm.skdc" //nolint:gosec
	CredentialRefreshMethod_PKCS10_REQUEST_TO_CA                  CredentialRefreshMethod = "oic.sec.crm.pk10" //nolint:gosec
)

type CredentialOptionalData struct {
	DataInternal interface{}                    `json:"data" yaml:"data"`
	Encoding     CredentialOptionalDataEncoding `json:"encoding" yaml:"encoding"`
	IsRevoked    bool                           `json:"revstat" yaml:"isRevoked,omitempty"`
}

func toByte(v interface{}) []byte {
	if v == nil {
		return nil
	}
	switch v := v.(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	}
	return nil
}

func (c CredentialOptionalData) Data() []byte {
	return toByte(c.DataInternal)
}

const (
	dataEncoding_JWT    string = "oic.sec.encoding.jwt"
	dataEncoding_CWT    string = "oic.sec.encoding.cwt"
	dataEncoding_BASE64 string = "oic.sec.encoding.base64"
	dataEncoding_URI    string = "oic.sec.encoding.uri"
	dataEncoding_HANDLE string = "oic.sec.encoding.handle"
	dataEncoding_RAW    string = "oic.sec.encoding.raw"
)

type CredentialOptionalDataEncoding string

const (
	CredentialOptionalDataEncoding_JWT    CredentialOptionalDataEncoding = CredentialOptionalDataEncoding(dataEncoding_JWT)
	CredentialOptionalDataEncoding_CWT    CredentialOptionalDataEncoding = CredentialOptionalDataEncoding(dataEncoding_CWT)
	CredentialOptionalDataEncoding_BASE64 CredentialOptionalDataEncoding = CredentialOptionalDataEncoding(dataEncoding_BASE64)
	CredentialOptionalDataEncoding_PEM    CredentialOptionalDataEncoding = CredentialOptionalDataEncoding(csr.CertificateEncoding_PEM)
	CredentialOptionalDataEncoding_DER    CredentialOptionalDataEncoding = CredentialOptionalDataEncoding(csr.CertificateEncoding_DER) // iotivity-lite doesn't support it
	CredentialOptionalDataEncoding_RAW    CredentialOptionalDataEncoding = CredentialOptionalDataEncoding(dataEncoding_RAW)
)

type CredentialPrivateData struct {
	DataInternal interface{}                   `json:"data"`
	Encoding     CredentialPrivateDataEncoding `json:"encoding"`
	Handle       int                           `json:"handle,omitempty"`
}

func (c CredentialPrivateData) Data() []byte {
	return toByte(c.DataInternal)
}

type CredentialPrivateDataEncoding string

const (
	CredentialPrivateDataEncoding_JWT    CredentialPrivateDataEncoding = CredentialPrivateDataEncoding(dataEncoding_JWT)
	CredentialPrivateDataEncoding_CWT    CredentialPrivateDataEncoding = CredentialPrivateDataEncoding(dataEncoding_CWT)
	CredentialPrivateDataEncoding_BASE64 CredentialPrivateDataEncoding = CredentialPrivateDataEncoding(dataEncoding_BASE64)
	CredentialPrivateDataEncoding_URI    CredentialPrivateDataEncoding = CredentialPrivateDataEncoding(dataEncoding_URI)
	CredentialPrivateDataEncoding_HANDLE CredentialPrivateDataEncoding = CredentialPrivateDataEncoding(dataEncoding_HANDLE)
	CredentialPrivateDataEncoding_RAW    CredentialPrivateDataEncoding = CredentialPrivateDataEncoding(dataEncoding_RAW)
)

type CredentialPublicData struct {
	DataInternal interface{}                  `json:"data" yaml:"data"`
	Encoding     CredentialPublicDataEncoding `json:"encoding" yaml:"encoding"`
}

func (c CredentialPublicData) Data() []byte {
	return toByte(c.DataInternal)
}

type CredentialPublicDataEncoding string

const (
	CredentialPublicDataEncoding_JWT    CredentialPublicDataEncoding = CredentialPublicDataEncoding(dataEncoding_JWT)
	CredentialPublicDataEncoding_CWT    CredentialPublicDataEncoding = CredentialPublicDataEncoding(dataEncoding_CWT)
	CredentialPublicDataEncoding_BASE64 CredentialPublicDataEncoding = CredentialPublicDataEncoding(dataEncoding_BASE64)
	CredentialPublicDataEncoding_URI    CredentialPublicDataEncoding = CredentialPublicDataEncoding(dataEncoding_URI)
	CredentialPublicDataEncoding_PEM    CredentialPublicDataEncoding = CredentialPublicDataEncoding(csr.CertificateEncoding_PEM)
	CredentialPublicDataEncoding_DER    CredentialPublicDataEncoding = CredentialPublicDataEncoding(csr.CertificateEncoding_DER) // iotivity-lite doesn't support it
	CredentialPublicDataEncoding_RAW    CredentialPublicDataEncoding = CredentialPublicDataEncoding(dataEncoding_RAW)
)

type CredentialRoleID struct {
	Authority string `json:"authority,omitempty" yaml:"authority,omitempty"`
	Role      string `json:"role,omitempty" yaml:"role,omitempty"`
}

type CredentialResponse struct {
	ResourceOwner string       `json:"rowneruuid" yaml:"resourceOwner,omitempty"`
	Interfaces    []string     `json:"if,omitempty" yaml:"-"`
	ResourceTypes []string     `json:"rt,omitempty" yaml:"-"`
	Name          string       `json:"n,omitempty" yaml:"name,omitempty"`
	Credentials   []Credential `json:"creds" yaml:"creds"`
}

type CredentialUpdateRequest struct {
	ResourceOwner string       `json:"rowneruuid,omitempty"`
	Credentials   []Credential `json:"creds"`
}
