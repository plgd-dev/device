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
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/platform"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/require"
)

func TestDeviceGetResourcesIterator(t *testing.T) {
	ip := test.MustFindDeviceIP(test.DevsimName, test.IP4)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	defer func() {
		errC := c.Close()
		require.NoError(t, errC)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	dev, err := c.GetDeviceByIP(ctx, ip)
	require.NoError(t, err)
	require.NotEmpty(t, dev)
	allLinks, err := dev.GetResourceLinks(ctx, dev.GetEndpoints())
	require.NoError(t, err)

	// get some public resources only, so we don't have to own
	var links schema.ResourceLinks
	for _, link := range allLinks {
		if link.Href == platform.ResourceURI ||
			link.Href == device.ResourceURI {
			links = append(links, link)
		}
	}
	it := dev.GetResources(ctx, links)
	require.NotEmpty(t, it)

	var p platform.Platform
	var d device.Device
	i := 0
	for {
		var v interface{}
		if i < len(links) {
			if links[i].Href == platform.ResourceURI {
				v = &p
			}
			if links[i].Href == device.ResourceURI {
				v = &d
			}
		}

		if !it.Next(ctx, v) {
			break
		}
		i++
	}
	require.NoError(t, it.Err)
	require.Equal(t, len(links), i)
	require.NotEqual(t, platform.Platform{}, p)
	require.NotEqual(t, device.Device{}, d)
}
