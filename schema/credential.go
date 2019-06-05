package schema

import (
	"encoding/asn1"
	"fmt"
)

// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.cred.swagger.json

var ExtendedKeyUsage_IDENTITY_CERTIFICATE = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 44924, 1, 6}

type Credential struct {
	ID                      int                       `codec:"credid"`
	Type                    CredentialType            `codec:"credtype"`
	Usage                   CredentialUsage           `codec:"credusage"`
	SupportedRefreshMethods []CredentialRefreshMethod `codec:"crms,omitempty"`
	OptionalData            CredentialOptionalData    `codec:"optionaldata,omitempty"`
	Period                  string                    `codec:"period,omitempty"`
	PrivateData             CredentialPrivateData     `codec:"privatedata,omitempty"`
	PublicData              CredentialPublicData      `codec:"publicdata,omitempty"`
	RoleId                  CredentialRoleId          `codec:"roleid,omitempty"`
	Subject                 string                    `codec:"subjectuuid"`
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

func (c CredentialType) String() string {
	switch c {
	case CredentialType_EMPTY:
		return "EMPTY"
	case CredentialType_SYMMETRIC_PAIR_WISE:
		return "SYMMETRIC_PAIR_WISE"
	case CredentialType_SYMMETRIC_GROUP:
		return "SYMMETRIC_GROUP"
	case CredentialType_ASYMMETRIC_SIGNING:
		return "ASYMMETRIC_SIGNING"
	case CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE:
		return "ASYMMETRIC_SIGNING_WITH_CERTIFICATE"
	case CredentialType_PIN_OR_PASSWORD:
		return "PIN_OR_PASSWORD"
	case CredentialType_ASYMMETRIC_ENCRYPTION_KEY:
		return "ASYMMETRIC_ENCRYPTION_KEY"
	default:
		return fmt.Sprintf("unknown(%v)", uint8(c))
	}
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
	Data      string                         `codec:"data"`
	Encoding  CredentialOptionalDataEncoding `codec:"encoding"`
	IsRevoked bool                           `codec:"revstat"`
}

type CredentialOptionalDataEncoding string

const (
	CredentialOptionalDataEncoding_JWT    CredentialOptionalDataEncoding = "oic.sec.encoding.jwt"
	CredentialOptionalDataEncoding_CWT    CredentialOptionalDataEncoding = "oic.sec.encoding.cwt"
	CredentialOptionalDataEncoding_BASE64 CredentialOptionalDataEncoding = "oic.sec.encoding.base64"
	CredentialOptionalDataEncoding_PEM    CredentialOptionalDataEncoding = CredentialOptionalDataEncoding(CertificateEncoding_PEM)
	CredentialOptionalDataEncoding_DER    CredentialOptionalDataEncoding = CredentialOptionalDataEncoding(CertificateEncoding_DER)
	CredentialOptionalDataEncoding_RAW    CredentialOptionalDataEncoding = "oic.sec.encoding.raw"
)

type CredentialPrivateData struct {
	Data     string                        `codec:"data"`
	Encoding CredentialPrivateDataEncoding `codec:"encoding"`
	Handle   int                           `codec:"handle,omitempty"`
}

type CredentialPrivateDataEncoding string

const (
	CredentialPrivateDataEncoding_JWT    CredentialPrivateDataEncoding = "oic.sec.encoding.jwt"
	CredentialPrivateDataEncoding_CWT    CredentialPrivateDataEncoding = "oic.sec.encoding.cwt"
	CredentialPrivateDataEncoding_BASE64 CredentialPrivateDataEncoding = "oic.sec.encoding.base64"
	CredentialPrivateDataEncoding_URI    CredentialPrivateDataEncoding = "oic.sec.encoding.uri"
	CredentialPrivateDataEncoding_HANDLE CredentialPrivateDataEncoding = "oic.sec.encoding.handle"
	CredentialPrivateDataEncoding_RAW    CredentialPrivateDataEncoding = "oic.sec.encoding.raw"
)

type CredentialPublicData struct {
	Data     string                       `codec:"data"`
	Encoding CredentialPublicDataEncoding `codec:"encoding"`
}

type CredentialPublicDataEncoding string

const (
	CredentialPublicDataEncoding_JWT    CredentialPublicDataEncoding = "oic.sec.encoding.jwt"
	CredentialPublicDataEncoding_CWT    CredentialPublicDataEncoding = "oic.sec.encoding.cwt"
	CredentialPublicDataEncoding_BASE64 CredentialPublicDataEncoding = "oic.sec.encoding.base64"
	CredentialPublicDataEncoding_URI    CredentialPublicDataEncoding = "oic.sec.encoding.uri"
	CredentialPublicDataEncoding_PEM    CredentialPublicDataEncoding = CredentialPublicDataEncoding(CertificateEncoding_PEM)
	CredentialPublicDataEncoding_DER    CredentialPublicDataEncoding = CredentialPublicDataEncoding(CertificateEncoding_DER)
	CredentialPublicDataEncoding_RAW    CredentialPublicDataEncoding = "oic.sec.encoding.raw"
)

type CredentialRoleId struct {
	Authority string "authority"
	Role      string "role"
}

type CredentialResponse struct {
	ResourceOwner string       `codec:"rowneruuid"`
	Interfaces    []string     `codec:"if"`
	ResourceTypes []string     `codec:"rt"`
	Name          string       `codec:"n"`
	Credentials   []Credential `codec:"creds"`
}

type CredentialUpdateRequest struct {
	ResourceOwner string       `codec:"rowneruuid"`
	Credentials   []Credential `codec:"creds"`
}
