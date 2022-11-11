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
		errC := c.Close()
		require.NoError(t, errC)
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
		errC := c2.Close()
		require.NoError(t, errC)
	}()
	d, err := c2.GetDeviceByMulticast(ctx, c.DeviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	defer func() {
		errC := d.Close(ctx)
		require.NoError(t, errC)
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
		errC := c3.Close()
		require.NoError(t, errC)
	}()
	d, err = c3.GetDeviceByMulticast(ctx, c.DeviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	defer func() {
		errC := d.Close(ctx)
		require.NoError(t, errC)
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
		errC := c.Close()
		require.NoError(t, errC)
	}()
	c.SetUpTestDevice(t)

	pc, err := c.Provision(context.Background(), c.DeviceLinks)
	require.NoError(t, err)

	defer func() {
		errC := pc.Close(context.Background())
		require.NoError(t, errC)
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
