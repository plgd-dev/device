package core_test

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	ocf "github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/schema/acl"
	"github.com/plgd-dev/device/schema/cloud"
	"github.com/plgd-dev/device/test"
	"github.com/stretchr/testify/require"
)

func TestProvisioning(t *testing.T) {
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	c.SetUpTestDevice(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pc, err := c.Provision(ctx, c.DeviceLinks)
	require.NoError(t, err)

	require.NoError(t, pc.SetAccessControl(ctx, acl.AllPermissions, acl.TLSConnection, acl.AllResources...))

	derBlock, _ := pem.Decode(test.RootCACrt)
	require.NotEmpty(t, derBlock)
	ca, err := x509.ParseCertificate(derBlock.Bytes)
	require.NoError(t, err)

	err = pc.AddCertificateAuthority(ctx, "*", ca)
	require.NoError(t, err)

	err = pc.Close(ctx)
	require.NoError(t, err)

	cert := test.GenerateIdentityCert(Cert2Identity)
	require.NoError(t, err)
	c2, err := NewTestSecureClientWithCert(cert, false, false)
	require.NoError(t, err)
	d, err := c2.GetDeviceByMulticast(ctx, c.DeviceID, ocf.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	defer d.Close(ctx)
	eps := d.GetEndpoints()
	links, err := d.GetResourceLinks(ctx, eps)
	require.NoError(t, err)
	link, ok := links.GetResourceLink(test.TestResourceLightInstanceHref("1"))
	require.True(t, ok)
	err = d.GetResource(ctx, link, nil)
	require.NoError(t, err)
}

func TestSettingCloudResource(t *testing.T) {
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer c.Close()
	c.SetUpTestDevice(t)

	pc, err := c.Provision(context.Background(), c.DeviceLinks)
	require.NoError(t, err)

	defer func() {
		err = pc.Close(context.Background())
		require.NoError(t, err)
	}()

	require.NoError(t, pc.SetAccessControl(context.Background(), acl.AllPermissions, acl.TLSConnection, acl.AllResources...))

	r := cloud.ConfigurationUpdateRequest{
		AuthorizationProvider: "testAuthorizationProvider",
		URL:                   "testURL",
		AuthorizationCode:     "testAuthorizationCode",
	}
	err = pc.SetCloudResource(context.Background(), r)
	require.NoError(t, err)
}

var (
	Cert2Identity = "08987e91-1a08-495a-8b4c-ad3d413012d6"
)
