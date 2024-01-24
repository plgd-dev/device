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

func TestMakeSignInRequest(t *testing.T) {
	_, err := cloud.MakeSignInRequest("id", "uid", "")
	require.ErrorIs(t, err, cloud.ErrMissingAccessToken)

	req, err := cloud.MakeSignInRequest("id", "uid", "token")
	require.NoError(t, err)
	require.Equal(t, "id", req.DeviceID)
	require.Equal(t, "uid", req.UserID)
	require.Equal(t, "token", req.AccessToken)
	require.True(t, req.Login)
}
