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

package net

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	type data struct {
		maxMsgSize            uint32
		externalAddressesPort externalAddressesPort
	}
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		want    data
	}{
		{
			name:   "ValidConfig",
			config: &Config{ExternalAddresses: []string{"localhost:12345"}, MaxMessageSize: 1024},
			want: data{
				maxMsgSize: 1024,
				externalAddressesPort: externalAddressesPort{{
					host:    "localhost",
					port:    "12345",
					network: UDP4,
				}},
			},
		},
		{
			name:   "ValidConfigUDP6",
			config: &Config{ExternalAddresses: []string{"[::1]:12345"}, MaxMessageSize: 1024},
			want: data{
				maxMsgSize: 1024,
				externalAddressesPort: externalAddressesPort{{
					host:    "::1",
					port:    "12345",
					network: UDP6,
				}},
			},
		},
		{
			name:   "ValidConfigUDP4andUDP6",
			config: &Config{ExternalAddresses: []string{"localhost:12345", "[::1]:12345"}, MaxMessageSize: 1024},
			want: data{
				maxMsgSize: 1024,
				externalAddressesPort: externalAddressesPort{
					{
						host:    "localhost",
						port:    "12345",
						network: UDP4,
					},
					{
						host:    "::1",
						port:    "12345",
						network: UDP6,
					},
				},
			},
		},
		{
			name:   "ValidConfigWithDefaultMaxMessageSize",
			config: &Config{ExternalAddresses: []string{"localhost:12345"}},
			want: data{
				maxMsgSize: DefaultMaxMessageSize,
				externalAddressesPort: externalAddressesPort{{
					host:    "localhost",
					port:    "12345",
					network: UDP4,
				}},
			},
		},
		{
			name:    "MissingExternalAddress",
			config:  &Config{},
			wantErr: true,
		},
		{
			name:    "InvalidExternalAddress",
			config:  &Config{ExternalAddresses: []string{"invalid-address"}},
			wantErr: true,
		},
		{
			name:    "EmptyHostInExternalAddress",
			config:  &Config{ExternalAddresses: []string{":12345"}},
			wantErr: true,
		},
		{
			name:    "InvalidPortInExternalAddress",
			config:  &Config{ExternalAddresses: []string{"localhost:invalid"}},
			wantErr: true,
		},
		{
			name:    "PortGreaterThanMaxUint16",
			config:  &Config{ExternalAddresses: []string{"localhost:65536"}},
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
			require.Equal(t, tt.want.externalAddressesPort, tt.config.externalAddressesPort)
		})
	}
}
