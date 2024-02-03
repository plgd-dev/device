/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package credential_test

import (
	"crypto/x509"
	"testing"

	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/device/credential"
	"github.com/stretchr/testify/require"
)

func TestGetPool(t *testing.T) {
	getCAPool := func() []*x509.Certificate {
		return []*x509.Certificate{{}}
	}

	credPool1 := credential.MakeCAPool(nil, getCAPool)
	require.True(t, credPool1.IsValid())
	_, err := credPool1.GetPool()
	require.NoError(t, err)

	cloudCAPool1 := cloud.MakeCAPool(getCAPool, true)
	require.True(t, cloudCAPool1.IsValid())
	credPool2 := credential.MakeCAPool(cloudCAPool1, nil)
	require.True(t, credPool2.IsValid())
	_, err = credPool2.GetPool()
	require.NoError(t, err)

	credPool3 := credential.MakeCAPool(cloudCAPool1, getCAPool)
	require.True(t, credPool3.IsValid())
	_, err = credPool3.GetPool()
	require.NoError(t, err)

	credPool4 := credential.MakeCAPool(nil, nil)
	require.False(t, credPool4.IsValid())
}
