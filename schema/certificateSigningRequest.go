package schema

type CertificateSigningRequestResponse struct {
	Interfaces                []string            `json:"if"`
	ResourceTypes             []string            `json:"rt"`
	Name                      string              `json:"n"`
	InstanceId                string              `json:"id"`
	Encoding                  CertificateEncoding `json:"encoding"`
	CertificateSigningRequest interface{}         `json:"csr"`
}

func (c CertificateSigningRequestResponse) CSR() []byte {
	if c.CertificateSigningRequest == nil {
		return nil
	}
	switch v := c.CertificateSigningRequest.(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	}
	return nil
}

type CertificateEncoding string

const (
	CertificateEncoding_PEM CertificateEncoding = "oic.sec.encoding.pem"
	CertificateEncoding_DER CertificateEncoding = "oic.sec.encoding.der" // iotivity-lite doesn't support it
)
