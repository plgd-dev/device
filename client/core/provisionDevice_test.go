package core_test

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/schema/acl"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/require"
)

func TestProvisioning(t *testing.T) {
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	c.SetUpTestDevice(t)
	defer func() {
		errClose := c.Close()
		require.NoError(t, errClose)
	}()

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
	defer func() {
		err := c2.Close()
		require.NoError(t, err)
	}()
	d, err := c2.GetDeviceByMulticast(ctx, c.DeviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	defer func() {
		errClose := d.Close(ctx)
		require.NoError(t, errClose)
	}()
	eps := d.GetEndpoints()
	links, err := d.GetResourceLinks(ctx, eps)
	require.NoError(t, err)
	link, ok := links.GetResourceLink(test.TestResourceLightInstanceHref("1"))
	require.True(t, ok)
	err = d.GetResource(ctx, link, nil)
	require.NoError(t, err)

	c3, err := NewTestSecureClientWithCert(cert, true, false)
	require.NoError(t, err)
	defer func() {
		err := c3.Close()
		require.NoError(t, err)
	}()
	d, err = c3.GetDeviceByMulticast(ctx, c.DeviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	defer func() {
		errClose := d.Close(ctx)
		require.NoError(t, errClose)
	}()
	eps = d.GetEndpoints()
	links, err = d.GetResourceLinks(ctx, eps)
	require.NoError(t, err)
	link, ok = links.GetResourceLink(test.TestResourceLightInstanceHref("1"))
	require.True(t, ok)
	err = d.GetResource(ctx, link, nil)

	// DTLS is not supported, but TCP-TLS at the device doesn't support golang cipher suites
	require.NoError(t, err)
}

func TestSettingCloudResource(t *testing.T) {
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errClose := c.Close()
		require.NoError(t, errClose)
	}()
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

var Cert2Identity = "08987e91-1a08-495a-8b4c-ad3d413012d6"
