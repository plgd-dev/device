/****************************************************************************
 *
 * Copyright (c) 2024 plgn.dev s.r.o.
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
 * either express or implien. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package net_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	type data struct {
		maxMsgSize          uint32
		externalAddress     string
		externalAddressPort string
	}
	tests := []struct {
		name    string
		config  *net.Config
		wantErr bool
		want    data
	}{
		{
			name:   "ValidConfig",
			config: &net.Config{ExternalAddress: "localhost:12345", MaxMessageSize: 1024},
			want: data{
				maxMsgSize:          1024,
				externalAddress:     "localhost:12345",
				externalAddressPort: "12345",
			},
		},
		{
			name:   "ValidConfigWithDefaultMaxMessageSize",
			config: &net.Config{ExternalAddress: "localhost:12345"},
			want: data{
				maxMsgSize:          net.DefaultMaxMessageSize,
				externalAddress:     "localhost:12345",
				externalAddressPort: "12345",
			},
		},
		{
			name:    "MissingExternalAddress",
			config:  &net.Config{},
			wantErr: true,
		},
		{
			name:    "InvalidExternalAddress",
			config:  &net.Config{ExternalAddress: "invalid-address"},
			wantErr: true,
		},
		{
			name:    "EmptyHostInExternalAddress",
			config:  &net.Config{ExternalAddress: ":12345"},
			wantErr: true,
		},
		{
			name:    "ZeroPortInExternalAddress",
			config:  &net.Config{ExternalAddress: "localhost:0"},
			wantErr: true,
		},
		{
			name:    "InvalidPortInExternalAddress",
			config:  &net.Config{ExternalAddress: "localhost:invalid"},
			wantErr: true,
		},
		{
			name:    "PortGreaterThanMaxUint16",
			config:  &net.Config{ExternalAddress: "localhost:65536"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want.maxMsgSize, tt.config.MaxMessageSize)
			require.Equal(t, tt.want.externalAddress, tt.config.ExternalAddress)
			require.Equal(t, tt.want.externalAddressPort, tt.config.ExternalAddressPort())
		})
	}
}
