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

// Certificate Signing Request
// https://github.com/openconnectivityfoundation/security/blob/master/swagger2.0/oic.sec.csr.swagger.json
package csr

const (
	ResourceType = "oic.r.csr"
	ResourceURI  = "/oic/sec/csr"
)

type CertificateSigningRequestResponse struct {
	Interfaces                []string            `json:"if"`
	ResourceTypes             []string            `json:"rt"`
	Name                      string              `json:"n"`
	InstanceID                string              `json:"id"`
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
