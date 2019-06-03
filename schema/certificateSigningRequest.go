package schema

type CertificateSigningRequestResponse struct {
	Interfaces                []string            `codec:"if"`
	ResourceTypes             []string            `codec:"rt"`
	Name                      string              `codec:"n"`
	InstanceId                string              `codec:"id"`
	Encoding                  CertificateEncoding `codec:"encoding"`
	CertificateSigningRequest []byte              `codec:"csr"`
}

type CertificateEncoding string

const (
	CertificateEncoding_PEM CertificateEncoding = "oic.sec.encoding.pem"
	CertificateEncoding_DER CertificateEncoding = "oic.sec.encoding.der"
)
