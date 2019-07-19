package local_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	ocf "github.com/go-ocf/sdk/local"
	"github.com/stretchr/testify/require"
)

func setupSecureClient(t *testing.T) (*ocf.Client, *ocf.ManufacturerOTMClient) {
	mfgCert, err := tls.X509KeyPair(MfgCert, MfgKey)
	require.NoError(t, err)
	mfgTrustedCABlock, _ := pem.Decode(MfgTrustedCA)
	require.NotEmpty(t, mfgTrustedCABlock)
	mfgCa, err := x509.ParseCertificate(mfgTrustedCABlock.Bytes)
	require.NoError(t, err)

	identityIntermediateCABlock, _ := pem.Decode(IdentityIntermediateCA)
	require.NotEmpty(t, identityIntermediateCABlock)
	identityIntermediateCA, err := x509.ParseCertificates(identityIntermediateCABlock.Bytes)
	require.NoError(t, err)
	identityIntermediateCAKeyBlock, _ := pem.Decode(IdentityIntermediateCAKey)
	require.NotEmpty(t, identityIntermediateCAKeyBlock)
	identityIntermediateCAKey, err := x509.ParseECPrivateKey(identityIntermediateCAKeyBlock.Bytes)
	require.NoError(t, err)

	identityTrustedCABlock, _ := pem.Decode(IdentityTrustedCA)
	require.NotEmpty(t, identityTrustedCABlock)
	identityTrustedCA, err := x509.ParseCertificates(identityTrustedCABlock.Bytes)
	require.NoError(t, err)

	identityCert, err := tls.X509KeyPair(IdentityCert, IdentityKey)
	require.NoError(t, err)

	signer := ocf.NewBasicCertificateSigner(identityIntermediateCA, identityIntermediateCAKey, time.Hour*86400)

	otm := ocf.NewManufacturerOTMClient(mfgCert, mfgCa, signer, identityTrustedCA)
	require.NoError(t, err)

	c := ocf.NewClient(ocf.WithTLS(&ocf.TLSConfig{
		GetCertificate: func() (tls.Certificate, error) {
			return identityCert, nil
		},
		GetCertificateAuthorities: func() ([]*x509.Certificate, error) {
			certs := make([]*x509.Certificate, 0, 8)
			certs = append(certs, identityTrustedCA...)
			certs = append(certs, mfgCa)
			return certs, nil
		},
	}))
	return c, otm
}

func TestClient_ownDevice(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name: "valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, otm := setupSecureClient(t)
			deviceId := testGetDeviceID(t, c, true)
			require := require.New(t)
			timeout, cancelTimeout := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancelTimeout()

			device, _, err := c.GetDevice(timeout, deviceId)
			require.NoError(err)
			defer device.Close(timeout)

			err = device.Own(timeout, otm)
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
			err = device.Disown(timeout)
			if tt.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}

var (
	CertIdentity = "b5a2a42e-b285-42f1-a36b-034c8fc8efd5"

	MfgCert = []byte(`-----BEGIN CERTIFICATE-----
MIIB9zCCAZygAwIBAgIRAOwIWPAt19w7DswoszkVIEIwCgYIKoZIzj0EAwIwEzER
MA8GA1UEChMIVGVzdCBPUkcwHhcNMTkwNTAyMjAwNjQ4WhcNMjkwMzEwMjAwNjQ4
WjBHMREwDwYDVQQKEwhUZXN0IE9SRzEyMDAGA1UEAxMpdXVpZDpiNWEyYTQyZS1i
Mjg1LTQyZjEtYTM2Yi0wMzRjOGZjOGVmZDUwWTATBgcqhkjOPQIBBggqhkjOPQMB
BwNCAAQS4eiM0HNPROaiAknAOW08mpCKDQmpMUkywdcNKoJv1qnEedBhWne7Z0jq
zSYQbyqyIVGujnI3K7C63NRbQOXQo4GcMIGZMA4GA1UdDwEB/wQEAwIDiDAzBgNV
HSUELDAqBggrBgEFBQcDAQYIKwYBBQUHAwIGCCsGAQUFBwMBBgorBgEEAYLefAEG
MAwGA1UdEwEB/wQCMAAwRAYDVR0RBD0wO4IJbG9jYWxob3N0hwQAAAAAhwR/AAAB
hxAAAAAAAAAAAAAAAAAAAAAAhxAAAAAAAAAAAAAAAAAAAAABMAoGCCqGSM49BAMC
A0kAMEYCIQDuhl6zj6gl2YZbBzh7Th0uu5izdISuU/ESG+vHrEp7xwIhANCA7tSt
aBlce+W76mTIhwMFXQfyF3awWIGjOcfTV8pU
-----END CERTIFICATE-----
`)

	MfgKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIMPeADszZajrkEy4YvACwcbR0pSdlKG+m8ALJ6lj/ykdoAoGCCqGSM49
AwEHoUQDQgAEEuHojNBzT0TmogJJwDltPJqQig0JqTFJMsHXDSqCb9apxHnQYVp3
u2dI6s0mEG8qsiFRro5yNyuwutzUW0Dl0A==
-----END EC PRIVATE KEY-----
`)

	MfgTrustedCA = []byte(`-----BEGIN CERTIFICATE-----
MIIBaTCCAQ+gAwIBAgIQR33gIB75I7Vi/QnMnmiWvzAKBggqhkjOPQQDAjATMREw
DwYDVQQKEwhUZXN0IE9SRzAeFw0xOTA1MDIyMDA1MTVaFw0yOTAzMTAyMDA1MTVa
MBMxETAPBgNVBAoTCFRlc3QgT1JHMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
xbwMaS8jcuibSYJkCmuVHfeV3xfYVyUq8Iroz7YlXaTayspW3K4hVdwIsy/5U+3U
vM/vdK5wn2+NrWy45vFAJqNFMEMwDgYDVR0PAQH/BAQDAgEGMBMGA1UdJQQMMAoG
CCsGAQUFBwMBMA8GA1UdEwEB/wQFMAMBAf8wCwYDVR0RBAQwAoIAMAoGCCqGSM49
BAMCA0gAMEUCIBWkxuHKgLSp6OXDJoztPP7/P5VBZiwLbfjTCVRxBvwWAiEAnzNu
6gKPwtKmY0pBxwCo3NNmzNpA6KrEOXE56PkiQYQ=
-----END CERTIFICATE-----		
`)
	MfgTrustedCAKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICzfC16AqtSv3wt+qIbrgM8dTqBhHANJhZS5xCpH6P2roAoGCCqGSM49
AwEHoUQDQgAExbwMaS8jcuibSYJkCmuVHfeV3xfYVyUq8Iroz7YlXaTayspW3K4h
VdwIsy/5U+3UvM/vdK5wn2+NrWy45vFAJg==
-----END EC PRIVATE KEY-----	
`)

	IdentityTrustedCA = []byte(`-----BEGIN CERTIFICATE-----
MIIBaDCCAQ6gAwIBAgIRANpzWRKheR25RH0CgYYwLzQwCgYIKoZIzj0EAwIwETEP
MA0GA1UEAxMGUm9vdENBMCAXDTE5MDcxOTEzMTA1M1oYDzIxMTkwNjI1MTMxMDUz
WjARMQ8wDQYDVQQDEwZSb290Q0EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAASQ
TLfEiNgEfqyWmtW1RV9UKgxsMddrNlYFt/+ZpqaJpBQ+hvtGwJenLEv5jzeEcMXr
gOR4EwjjJSzELk6IibC+o0UwQzAOBgNVHQ8BAf8EBAMCAQYwEwYDVR0lBAwwCgYI
KwYBBQUHAwEwDwYDVR0TAQH/BAUwAwEB/zALBgNVHREEBDACggAwCgYIKoZIzj0E
AwIDSAAwRQIhAOUfsOKyjIgYmDd2G46ge+PEPAZ9DS67Q5RjJvLk/lf3AiA6yMxJ
msmj2nz8VeEkxpKq3gYwJUdJ9jMklTzP+Dcenw==
-----END CERTIFICATE-----
`)
	IdentityTrustedCAKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIFe+pAuS4dEt1gRZ6Mq1xrgkEHxL191AFzEsNNvTEWOYoAoGCCqGSM49
AwEHoUQDQgAEkEy3xIjYBH6slprVtUVfVCoMbDHXazZWBbf/maamiaQUPob7RsCX
pyxL+Y83hHDF64DkeBMI4yUsxC5OiImwvg==
-----END EC PRIVATE KEY-----
`)
	IdentityIntermediateCA = []byte(`-----BEGIN CERTIFICATE-----
MIIBbzCCARagAwIBAgIRAKfBOtAzyvyToyIf7oceyqwwCgYIKoZIzj0EAwIwETEP
MA0GA1UEAxMGUm9vdENBMCAXDTE5MDcxOTEzMTExNloYDzIxMTkwNjI1MTMxMTE2
WjAZMRcwFQYDVQQDEw5JbnRlcm1lZGlhdGVDQTBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABFJttL0tCktK++BB7e8r6WVXzQoXnZ+LWoo6kiBPs+xqVLZ3i0m6GQhg
/mCO/r6V/LfMDC+RtUj/WI/HRloVlFijRTBDMA4GA1UdDwEB/wQEAwIBBjATBgNV
HSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MAsGA1UdEQQEMAKCADAK
BggqhkjOPQQDAgNHADBEAiAJ76rryZcwF3lAzxJRmXc2Twh07Z7F8RrySv2DzW31
kAIgD6bRa6nyIw3q3MA/xDNF+SAj+pxLpB7TncnCHDYt0zs=
-----END CERTIFICATE-----
`)
	IdentityIntermediateCAKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEILN09HNEfbSgC2NCydiEaQhG8AiWdlgNsp5WRn4seB6woAoGCCqGSM49
AwEHoUQDQgAEUm20vS0KS0r74EHt7yvpZVfNChedn4taijqSIE+z7GpUtneLSboZ
CGD+YI7+vpX8t8wML5G1SP9Yj8dGWhWUWA==
-----END EC PRIVATE KEY-----
`)
	IdentityCert = []byte(`-----BEGIN CERTIFICATE-----
MIIBsTCCAVagAwIBAgIQOajyvg4xzmcTJlet709fyzAKBggqhkjOPQQDAjAZMRcw
FQYDVQQDEw5JbnRlcm1lZGlhdGVDQTAgFw0xOTA3MTkxMzExNTBaGA8yMTE5MDYy
NTEzMTE1MFowNDEyMDAGA1UEAxMpdXVpZDowMDAwMDAwMC0wMDAwLTAwMDAtMDAw
MC0wMDAwMDAwMDAwMDEwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQyUHCFDhO2
IOPiBF7D+eVDvqKUqXoHFlDO9me4pU8c0qQJN48Xy4uDg0DHg6gjr1YTnw2iSQ+M
ZIM59IInUoW9o2MwYTAOBgNVHQ8BAf8EBAMCA4gwMwYDVR0lBCwwKgYIKwYBBQUH
AwEGCCsGAQUFBwMCBggrBgEFBQcDAQYKKwYBBAGC3nwBBjAMBgNVHRMBAf8EAjAA
MAwGA1UdEQQFMAOCATowCgYIKoZIzj0EAwIDSQAwRgIhAJJJWDUhURrzBMNbkB+t
zSLPsM6g1FqO56gWDqoddlQ1AiEAj8Y+bYepeSUwM7nxKzerWMq3oeVp3SHVlgqv
PLg5J0Q=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIBbzCCARagAwIBAgIRAKfBOtAzyvyToyIf7oceyqwwCgYIKoZIzj0EAwIwETEP
MA0GA1UEAxMGUm9vdENBMCAXDTE5MDcxOTEzMTExNloYDzIxMTkwNjI1MTMxMTE2
WjAZMRcwFQYDVQQDEw5JbnRlcm1lZGlhdGVDQTBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABFJttL0tCktK++BB7e8r6WVXzQoXnZ+LWoo6kiBPs+xqVLZ3i0m6GQhg
/mCO/r6V/LfMDC+RtUj/WI/HRloVlFijRTBDMA4GA1UdDwEB/wQEAwIBBjATBgNV
HSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MAsGA1UdEQQEMAKCADAK
BggqhkjOPQQDAgNHADBEAiAJ76rryZcwF3lAzxJRmXc2Twh07Z7F8RrySv2DzW31
kAIgD6bRa6nyIw3q3MA/xDNF+SAj+pxLpB7TncnCHDYt0zs=
-----END CERTIFICATE-----
`)
	IdentityKey = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIMnl+B0/b8WYp2M3LbT5Kz5cs9/8LTIOs6TzacXqaVwnoAoGCCqGSM49
AwEHoUQDQgAEMlBwhQ4TtiDj4gRew/nlQ76ilKl6BxZQzvZnuKVPHNKkCTePF8uL
g4NAx4OoI69WE58NokkPjGSDOfSCJ1KFvQ==
-----END EC PRIVATE KEY-----
`)
)
