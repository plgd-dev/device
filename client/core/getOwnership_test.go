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

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/client/core/otm"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOwnership(t *testing.T) {
	secureDeviceID := test.MustFindDeviceByName(test.DevsimName)
	c, err := NewTestSecureClient()
	require.NoError(t, err)
	signer, err := test.NewTestSigner()
	require.NoError(t, err)
	defer func() {
		errClose := c.Close()
		require.NoError(t, errClose)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	deviceID := secureDeviceID
	device, err := c.GetDeviceByMulticast(ctx, deviceID, core.DefaultDiscoveryConfiguration())
	require.NoError(t, err)
	defer func() {
		errClose := device.Close(ctx)
		require.NoError(t, errClose)
	}()
	eps := device.GetEndpoints()
	links, err := device.GetResourceLinks(ctx, eps)
	require.NoError(t, err)

	// before own
	got, err := device.GetOwnership(ctx, links)
	require.NoError(t, err)
	assert.False(t, got.Owned)

	err = device.Own(ctx, links, []otm.Client{c.mfgOtm}, core.WithSetupCertificates(signer.Sign))
	require.NoError(t, err)

	// after own
	links, err = device.GetResourceLinks(ctx, eps)
	require.NoError(t, err)
	got, err = device.GetOwnership(ctx, links)
	require.NoError(t, err)
	assert.True(t, got.Owned)
	assert.Equal(t, CertIdentity, got.OwnerID)

	err = device.Disown(ctx, links)
	require.NoError(t, err)
}
