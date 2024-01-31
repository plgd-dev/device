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

package cloud_test

import (
	"crypto/x509"
	"testing"

	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/stretchr/testify/require"
)

func TestGetPool(t *testing.T) {
	getCAPool := func() []*x509.Certificate {
		return []*x509.Certificate{{}}
	}
	caPool1 := cloud.MakeCAPool(getCAPool, true)
	require.True(t, caPool1.IsValid())
	_, err := caPool1.GetPool()
	require.NoError(t, err)

	caPool2 := cloud.MakeCAPool(getCAPool, false)
	require.True(t, caPool2.IsValid())
	_, err = caPool2.GetPool()
	require.NoError(t, err)
}
