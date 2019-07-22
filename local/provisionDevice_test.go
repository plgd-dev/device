package local_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/go-ocf/sdk/schema"
	"github.com/go-ocf/sdk/schema/acl"
	"github.com/stretchr/testify/require"
)

func TestProvisioning(t *testing.T) {
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	c.SetUpTestDevice(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pc, err := c.Provision(ctx)
	require.NoError(t, err)

	require.NoError(t, pc.SetAccessControl(ctx, acl.AllPermissions, acl.TLSConnection, acl.AllResources...))

	derBlock, _ := pem.Decode(Cert2PEMBlock)
	require.NotEmpty(t, derBlock)
	ca, err := x509.ParseCertificate(derBlock.Bytes)
	require.NoError(t, err)

	err = pc.AddCertificateAuthority(ctx, "*", ca)
	require.NoError(t, err)

	err = pc.Close(ctx)
	require.NoError(t, err)

	cert, err := tls.X509KeyPair(Cert2PEMBlock, Cert2KeyPEMBlock)
	require.NoError(t, err)
	c2, err := NewTestSecureClientWithCert(cert)
	require.NoError(t, err)
	d, _, err := c2.GetDevice(ctx, c.DeviceID)
	require.NoError(t, err)
	defer d.Close(ctx)
	err = d.GetResource(ctx, "/light/1", nil)
	require.NoError(t, err)
}

func TestSettingCloudResource(t *testing.T) {
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()
	c.SetUpTestDevice(t)

	pc, err := c.Provision(context.Background())
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
MIIBijCCAS+gAwIBAgIRANepL9IJ9CJvWmz6m7HeJYEwCgYIKoZIzj0EAwIwGTEX
MBUGA1UEAxMOSW50ZXJtZWRpYXRlQ0EwHhcNMTkwNzIyMTkxNTU5WhcNMjkwNzE5
MTkxNTU5WjA0MTIwMAYDVQQDEyl1dWlkOjA4OTg3ZTkxLTFhMDgtNDk1YS04YjRj
LWFkM2Q0MTMwMTJkNjBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABPH5HyV14zH9
0wNKRBA/jG+N/xKFBZBRE3sSo+hq+uRWn8dDKUwvmfHGjrDzQajPEMrTw2ildeg2
6VeRBStJPVmjPTA7MA4GA1UdDwEB/wQEAwIDiDApBgNVHSUEIjAgBggrBgEFBQcD
AgYIKwYBBQUHAwEGCisGAQQBgt58AQYwCgYIKoZIzj0EAwIDSQAwRgIhAOkwULv5
a1xDSK03d0oW+SsN7dKME63WXP06DAx930pcAiEAmQ3PgepI681TzqgwipNo1T/7
cQKFUWOh0HnFvnePsE0=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIBczCCARmgAwIBAgIRANntjEpzu9krzL0EG6fcqqgwCgYIKoZIzj0EAwIwETEP
MA0GA1UEAxMGUm9vdENBMCAXDTE5MDcxOTIwMzczOVoYDzIxMTkwNjI1MjAzNzM5
WjAZMRcwFQYDVQQDEw5JbnRlcm1lZGlhdGVDQTBZMBMGByqGSM49AgEGCCqGSM49
AwEHA0IABKw1/6WHFcWtw67hH5DzoZvHgA0suC6IYLKms4IP/pds9wU320eDaENo
5860TOyKrGn7vW/cj/OVe2Dzr4KSFVijSDBGMA4GA1UdDwEB/wQEAwIBBjATBgNV
HSUEDDAKBggrBgEFBQcDATASBgNVHRMBAf8ECDAGAQH/AgEAMAsGA1UdEQQEMAKC
ADAKBggqhkjOPQQDAgNIADBFAiEAgPtnYpgwxmPhN0Mo8VX582RORnhcdSHMzFjh
P/li1WwCIFVVWBOrfBnTt7A6UfjP3ljAyHrJERlMauQR+tkD/aqm
-----END CERTIFICATE-----
`)

	Cert2KeyPEMBlock = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIFq52fGiTG896EiR6vOC6GZbBFJjXFW2vHRmukzxX+U2oAoGCCqGSM49
AwEHoUQDQgAE8fkfJXXjMf3TA0pEED+Mb43/EoUFkFETexKj6Gr65Fafx0MpTC+Z
8caOsPNBqM8QytPDaKV16DbpV5EFK0k9WQ==
-----END EC PRIVATE KEY-----
`)
)
