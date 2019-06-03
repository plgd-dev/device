package schema

import (
	"fmt"
)


// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.cred.swagger.json

type Credential struct {
	Id string `codec:"credid"`
	Type CredentialType `codec:"credtype"`
	Update CredentialUsage `codec:"credusage"`
}

type CredentialResponse struct {
	ResourceOwner                 string   `codec:"rowneruuid"`
	Interfaces                    []string `codec:"if"`
	ResourceTypes                 []string `codec:"rt"`
	Name                          string   `codec:"n"`

}

type CredentialType uint8

const (
	CredentialType_EMPTY               CredentialType = 0
	CredentialType_SYMMETRIC_PAIR_WISE CredentialType =  1 << iota
	CredentialType_SYMMETRIC_GROUP
	CredentialType_ASYMMETRIC_SIGNING
	CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE
	CredentialType_PIN_OR_PASSWORD
	CredentialType_ASYMMETRIC_ENCRYPTION_KEY
)

func (c CredentialType) String() string {
	switch (c) {
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
	CredentialUsage_TRUST_CA CredentialUsage = "oic.sec.cred.trustca"
	CredentialUsage_CERT CredentialUsage = "oic.sec.cred.cert"
	CredentialUsage_ROLE_CERT CredentialUsage = "oic.sec.cred.rolecert"
	CredentialUsage_MFG_TRUST_CA CredentialUsage = "oic.sec.cred.mfgtrustca"
	CredentialUsage_MFG_CERT CredentialUsage ="oic.sec.cred.mfgcert"
)

func (c CredentialUsage) String() string {
	switch (c) {
	case CredentialUsage_TRUST_CA:
		return "TRUST_CA"
	case CredentialUsage_CERT:
		return "CERT"
	case CredentialUsage_ROLE_CERT:
		return "ROLE_CERT"
	case CredentialUsage_MFG_TRUST_CA:
		return "MFG_TRUST_CA"
	case CredentialUsage_MFG_CERT:
		return "MFG_CERT"
	default:
		return fmt.Sprintf("unknown(%v)", string(c))
	}
}