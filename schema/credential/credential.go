// Credential
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.cred.swagger.json
package credential

import (
	"fmt"
	"strings"

	"github.com/plgd-dev/device/schema/csr"
)

const (
	ResourceType = "oic.r.cred"
	ResourceURI  = "/oic/sec/cred"
)

type Credential struct {
	ID                      int                       `json:"credid,omitempty"`
	Type                    CredentialType            `json:"credtype"`
	Subject                 string                    `json:"subjectuuid"`
	Usage                   CredentialUsage           `json:"credusage,omitempty"`
	SupportedRefreshMethods []CredentialRefreshMethod `json:"crms,omitempty"`
	OptionalData            *CredentialOptionalData   `json:"optionaldata,omitempty"`
	Period                  string                    `json:"period,omitempty"`
	PrivateData             *CredentialPrivateData    `json:"privatedata,omitempty"`
	PublicData              *CredentialPublicData     `json:"publicdata,omitempty"`
	RoleID                  *CredentialRoleID         `json:"roleid,omitempty"`
	Tag                     string                    `json:"tag,omitempty"`
}

type CredentialType uint8

const (
	CredentialType_EMPTY                               CredentialType = 0
	CredentialType_SYMMETRIC_PAIR_WISE                 CredentialType = 1
	CredentialType_SYMMETRIC_GROUP                     CredentialType = 2
	CredentialType_ASYMMETRIC_SIGNING                  CredentialType = 4
	CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE CredentialType = 8
	CredentialType_PIN_OR_PASSWORD                     CredentialType = 16
	CredentialType_ASYMMETRIC_ENCRYPTION_KEY           CredentialType = 32
)

func (s CredentialType) String() string {
	res := make([]string, 0, 7)
	if s.Has(CredentialType_EMPTY) {
		res = append(res, "EMPTY")
		s &^= CredentialType_EMPTY
	}
	if s.Has(CredentialType_SYMMETRIC_PAIR_WISE) {
		res = append(res, "SYMMETRIC_PAIR_WISE")
		s &^= CredentialType_SYMMETRIC_PAIR_WISE
	}
	if s.Has(CredentialType_SYMMETRIC_GROUP) {
		res = append(res, "SYMMETRIC_GROUP")
		s &^= CredentialType_SYMMETRIC_GROUP
	}
	if s.Has(CredentialType_ASYMMETRIC_SIGNING) {
		res = append(res, "ASYMMETRIC_SIGNING")
		s &^= CredentialType_ASYMMETRIC_SIGNING
	}
	if s.Has(CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE) {
		res = append(res, "ASYMMETRIC_SIGNING_WITH_CERTIFICATE")
		s &^= CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE
	}
	if s.Has(CredentialType_PIN_OR_PASSWORD) {
		res = append(res, "PIN_OR_PASSWORD")
		s &^= CredentialType_PIN_OR_PASSWORD
	}
	if s.Has(CredentialType_ASYMMETRIC_ENCRYPTION_KEY) {
		res = append(res, "ASYMMETRIC_ENCRYPTION_KEY")
		s &^= CredentialType_ASYMMETRIC_ENCRYPTION_KEY
	}
	if s != 0 {
		res = append(res, fmt.Sprintf("unknown(%v)", int(s)))
	}
	return strings.Join(res, "|")
}

// Has returns true if the flag is set.
func (b CredentialType) Has(flag CredentialType) bool {
	return b&flag != 0
}

type CredentialUsage string

const (
	CredentialUsage_TRUST_CA     CredentialUsage = "oic.sec.cred.trustca"
	CredentialUsage_CERT         CredentialUsage = "oic.sec.cred.cert"
	CredentialUsage_ROLE_CERT    CredentialUsage = "oic.sec.cred.rolecert"
	CredentialUsage_MFG_TRUST_CA CredentialUsage = "oic.sec.cred.mfgtrustca"
	CredentialUsage_MFG_CERT     CredentialUsage = "oic.sec.cred.mfgcert"
)

type CredentialRefreshMethod string

const (
	CredentialRefreshMethod_PROVISION_SERVICE                     CredentialRefreshMethod = "oic.sec.crm.pro"
	CredentialRefreshMethod_KEY_AGREEMENT_PROTOCOL_AND_RANDOM_PIN CredentialRefreshMethod = "oic.sec.crm.psk"
	CredentialRefreshMethod_KEY_AGREEMENT_PROTOCOL                CredentialRefreshMethod = "oic.sec.crm.rdp"
	CredentialRefreshMethod_KEY_DISTRIBUTION_SERVICE              CredentialRefreshMethod = "oic.sec.crm.skdc"
	CredentialRefreshMethod_PKCS10_REQUEST_TO_CA                  CredentialRefreshMethod = "oic.sec.crm.pk10"
)

type CredentialOptionalData struct {
	DataInternal interface{}                    `json:"data"`
	Encoding     CredentialOptionalDataEncoding `json:"encoding"`
	IsRevoked    bool                           `json:"revstat"`
}

func (c CredentialOptionalData) Data() []byte {
	if c.DataInternal == nil {
		return nil
	}
	switch v := c.DataInternal.(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	}
	return nil
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
	if c.DataInternal == nil {
		return nil
	}
	switch v := c.DataInternal.(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	}
	return nil
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
	DataInternal interface{}                  `json:"data"`
	Encoding     CredentialPublicDataEncoding `json:"encoding"`
}

func (c CredentialPublicData) Data() []byte {
	if c.DataInternal == nil {
		return nil
	}
	switch v := c.DataInternal.(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	}
	return nil
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
	Authority string `json:"authority,omitempty"`
	Role      string `json:"role,omitempty"`
}

type CredentialResponse struct {
	ResourceOwner string       `json:"rowneruuid"`
	Interfaces    []string     `json:"if"`
	ResourceTypes []string     `json:"rt"`
	Name          string       `json:"n"`
	Credentials   []Credential `json:"creds"`
}

type CredentialUpdateRequest struct {
	ResourceOwner string       `json:"rowneruuid,omitempty"`
	Credentials   []Credential `json:"creds"`
}
