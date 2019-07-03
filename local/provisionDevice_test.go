package local_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/schema/acl"
	"github.com/stretchr/testify/require"
)

func TestProvisioning(t *testing.T) {
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()
	c.SetUpTestDevice(t)

	pc, err := c.ProvisionDevice(context.Background(), c.DeviceID)
	require.NoError(t, err)

	require.NoError(t, pc.SetAccessControl(context.Background(), acl.AllPermissions, acl.TLSConnection, acl.AllResources...))

	derBlock, _ := pem.Decode(Cert2PEMBlock)
	require.NotEmpty(t, derBlock)
	ca, err := x509.ParseCertificate(derBlock.Bytes)
	require.NoError(t, err)

	err = pc.AddCertificateAuthority(context.Background(), "*", ca)
	require.NoError(t, err)

	err = pc.Close(context.Background())
	require.NoError(t, err)

	cert, err := tls.X509KeyPair(Cert2PEMBlock, Cert2KeyPEMBlock)
	require.NoError(t, err)
	c2, err := NewTestSecureClientWithCert(cert)
	require.NoError(t, err)
	d, err := c2.GetDevice(context.Background(), c.DeviceID)
	require.NoError(t, err)
	err = d.GetResource(context.Background(), c.DeviceID, "/light/1", nil)
	require.NoError(t, err)
}

func TestSettingCloudResource(t *testing.T) {
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()
	c.SetUpTestDevice(t)

	pc, err := c.ProvisionDevice(context.Background(), c.DeviceID)
	require.NoError(t, err)

	defer func() {
		err = pc.Close(context.Background())
		require.NoError(t, err)
	}()

	r := schema.CloudUpdateRequest{
		AuthorizationProvider: "testAuthorizationProvider",
		URL:                   "testURL",
		AuthorizationCode:     "testAuthorizationCode",
	}
	err = pc.SetCloudResource(context.Background(), r)
	require.NoError(t, err)
}

var (
	Cert2Identity = "08987e91-1a08-495a-8b4c-ad3d413012d6"

	Cert2PEMBlock = []byte(`-----BEGIN CERTIFICATE-----
MIIBxDCCAWugAwIBAgIQO9EpFwN72HoRq8arCO8wzTAKBggqhkjOPQQDAjATMREw
DwYDVQQKEwhUZXN0IE9SRzAgFw0xOTA2MTcxMTQ5MzFaGA8yMTMzMDcxNjAzNDkz
MVowRzERMA8GA1UEChMIVGVzdCBPUkcxMjAwBgNVBAMTKXV1aWQ6MDg5ODdlOTEt
MWEwOC00OTVhLThiNGMtYWQzZDQxMzAxMmQ2MFkwEwYHKoZIzj0CAQYIKoZIzj0D
AQcDQgAEQpFpqd6B89V9YondW6fBtbvoWce/IdvQI8tmgkbq/U1hamNlzKeGCVL/
My/gjhS4jvtyB6mSyfLkH/3hlcnp1KNrMGkwDgYDVR0PAQH/BAQDAgOIMDMGA1Ud
JQQsMCoGCCsGAQUFBwMBBggrBgEFBQcDAgYIKwYBBQUHAwEGCisGAQQBgt58AQYw
DAYDVR0TAQH/BAIwADAUBgNVHREEDTALgglsb2NhbGhvc3QwCgYIKoZIzj0EAwID
RwAwRAIgCl553pNli4l6EUCnzBg/KJvwB1B/7xj9lCaN1tKQEo0CIH1mddc+PtRe
O+bWHqY22wN6qGAAleg3yh60R66RIHEu
-----END CERTIFICATE-----
`)

	Cert2KeyPEMBlock = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIwkxSDWHvqkVRjy4EmuoucZqWFL0mzUo/Rjbt6V0n4ooAoGCCqGSM49
AwEHoUQDQgAEQpFpqd6B89V9YondW6fBtbvoWce/IdvQI8tmgkbq/U1hamNlzKeG
CVL/My/gjhS4jvtyB6mSyfLkH/3hlcnp1A==
-----END EC PRIVATE KEY-----
`)
)
