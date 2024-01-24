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
	"testing"

	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/stretchr/testify/require"
)

func TestMakeSignUpRequest(t *testing.T) {
	_, err := cloud.MakeSignUpRequest("id", "", "provider")
	require.ErrorIs(t, err, cloud.ErrMissingAuthorizationCode)

	_, err = cloud.MakeSignUpRequest("id", "code", "")
	require.ErrorIs(t, err, cloud.ErrMissingAuthorizationProvider)

	req, err := cloud.MakeSignUpRequest("id", "code", "provider")
	require.NoError(t, err)
	require.Equal(t, "id", req.DeviceID)
	require.Equal(t, "code", req.AuthorizationCode)
	require.Equal(t, "provider", req.AuthorizationProvider)
}
