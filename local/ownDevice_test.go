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
	cert, err := tls.X509KeyPair(CertPEMBlock, KeyPEMBlock)
	require.NoError(t, err)
	derBlock, _ := pem.Decode(CARootPemBlock)
	require.NotEmpty(t, derBlock)
	ca, err := x509.ParseCertificate(derBlock.Bytes)
	require.NoError(t, err)
	derBlockKey, _ := pem.Decode(CARootKeyPemBlock)
	require.NotEmpty(t, derBlockKey)
	caKey, err := x509.ParseECPrivateKey(derBlockKey.Bytes)
	require.NoError(t, err)

	signer := ocf.NewBasicCertificateSigner(ca, caKey, time.Hour*86400)

	otm := ocf.NewManufacturerOTMClient(cert, ca, signer, []*x509.Certificate{ca})
	require.NoError(t, err)

	c := ocf.NewClient(ocf.WithTLS(&ocf.TLSConfig{
		GetCertificate: func() (tls.Certificate, error) {
			return cert, nil
		},
		GetCertificateAuthorities: func() ([]*x509.Certificate, error) {
			return []*x509.Certificate{ca}, nil
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

	CertPEMBlock = []byte(`-----BEGIN CERTIFICATE-----
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

	KeyPEMBlock = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIMPeADszZajrkEy4YvACwcbR0pSdlKG+m8ALJ6lj/ykdoAoGCCqGSM49
AwEHoUQDQgAEEuHojNBzT0TmogJJwDltPJqQig0JqTFJMsHXDSqCb9apxHnQYVp3
u2dI6s0mEG8qsiFRro5yNyuwutzUW0Dl0A==
-----END EC PRIVATE KEY-----
`)

	CARootPemBlock = []byte(`-----BEGIN CERTIFICATE-----
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
	CARootKeyPemBlock = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICzfC16AqtSv3wt+qIbrgM8dTqBhHANJhZS5xCpH6P2roAoGCCqGSM49
AwEHoUQDQgAExbwMaS8jcuibSYJkCmuVHfeV3xfYVyUq8Iroz7YlXaTayspW3K4h
VdwIsy/5U+3UvM/vdK5wn2+NrWy45vFAJg==
-----END EC PRIVATE KEY-----	
`)
)
