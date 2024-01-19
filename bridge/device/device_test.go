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

package device_test

import (
	"testing"

	"github.com/google/uuid"
	bridgeDevice "github.com/plgd-dev/device/v2/bridge/device"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       bridgeDevice.Config
		wantError bool
	}{
		{
			name: "Valid Configuration",
			cfg: bridgeDevice.Config{
				ProtocolIndependentID: uuid.New(),
				ID:                    uuid.New(),
				Name:                  "ValidName",
			},
		},
		{
			name: "Valid Configuration with empty ID and Name",
			cfg: bridgeDevice.Config{
				ProtocolIndependentID: uuid.New(),
			},
		},
		{
			name: "Invalid ProtocolIndependentID",
			cfg: bridgeDevice.Config{
				ID: uuid.New(),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
