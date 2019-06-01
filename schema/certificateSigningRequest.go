package schema

import "fmt"

type CertificateSigningRequestResponse struct {
	Interfaces                []string    `codec:"if"`
	ResourceTypes             []string    `codec:"rt"`
	Name                      string      `codec:"n"`
	InstanceId                string      `codec:"id"`
	Encoding                  CSREncoding `codec:"encoding"`
	CertificateSigningRequest []byte      `codec:"csr"`
}

type CSREncoding string

const (
	CSREncoding_PEM CSREncoding = "oic.sec.encoding.pem"
	CSREncoding_DER CSREncoding = "oic.sec.encoding.der"
)

func (s CSREncoding) String() string {
	switch s {
	case CSREncoding_PEM:
		return "PEM"
	case CSREncoding_DER:
		return "DER"
	default:
		return fmt.Sprintf("unknown %v", string(s))
	}
}
