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
	"sync"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/client/core/otm"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/require"
)

func TestClientGetDeviceByIPWithIP4(t *testing.T) {
	ip := test.MustFindDeviceIP(test.DevsimName, test.IP4)

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got, err := c.GetDeviceByIP(ctx, ip)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	links, err := got.GetResourceLinks(ctx, got.GetEndpoints())
	require.NoError(t, err)
	link, ok := links.GetResourceLink(platform.ResourceURI)
	require.True(t, ok)
	var v interface{}
	err = got.GetResource(ctx, link, &v)
	require.NoError(t, err)
}

func TestClientGetDeviceParallel(t *testing.T) {
	ip := test.MustFindDeviceIP(test.DevsimName, test.IP4)

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	count := 5
	numParallel := 10

	for i := 0; i < count; i++ {
		var wg sync.WaitGroup
		wg.Add(numParallel)
		for j := 0; j < numParallel; j++ {
			go func() {
				defer wg.Done()
				got, err := c.GetDeviceByIP(ctx, ip)
				require.NoError(t, err)
				require.NotEmpty(t, got)
				links, err := got.GetResourceLinks(ctx, got.GetEndpoints())
				require.NoError(t, err)
				link, ok := links.GetResourceLink(platform.ResourceURI)
				require.True(t, ok)
				var v interface{}
				err = got.GetResource(ctx, link, &v)
				require.NoError(t, err)
			}()
		}
		wg.Wait()
	}
}

func TestClientGetDeviceByIPWithIP6(t *testing.T) {
	ip := test.MustFindDeviceIP(test.DevsimName, test.IP6)

	c, err := NewTestSecureClient()
	require.NoError(t, err)
	signer, err := NewTestSigner()
	require.NoError(t, err)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got, err := c.GetDeviceByIP(ctx, ip)
	require.NoError(t, err)
	require.NotEmpty(t, got)
	links, err := got.GetResourceLinks(ctx, got.GetEndpoints())
	require.NoError(t, err)

	err = got.Own(ctx, links, []otm.Client{c.justWorksOtm}, core.WithSetupCertificates(signer.Sign))
	require.NoError(t, err)
	links, err = got.GetResourceLinks(ctx, got.GetEndpoints())
	require.NoError(t, err)
	defer func() {
		err := got.Disown(ctx, links)
		require.NoError(t, err)
	}()
}
