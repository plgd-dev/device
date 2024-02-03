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

package service_test

import (
	"testing"

	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/service"
	"github.com/stretchr/testify/require"
)

func TestCoAPConfigValidate(t *testing.T) {
	tests := []struct {
		name       string
		coapConfig service.CoAPConfig
		wantErr    bool
	}{
		{
			name:       "ValidCoAPConfig",
			coapConfig: service.CoAPConfig{ID: "test", Config: net.Config{ExternalAddresses: []string{"localhost:12345"}}},
		},
		{
			name:       "MissingID",
			coapConfig: service.CoAPConfig{Config: net.Config{ExternalAddresses: []string{"localhost:12345"}}},
			wantErr:    true,
		},
		{
			name:       "InvalidConfig",
			coapConfig: service.CoAPConfig{ID: "test", Config: net.Config{}},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.coapConfig.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAPIConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		apiConfig service.APIConfig
		wantErr   bool
	}{
		{
			name:      "ValidAPIConfig",
			apiConfig: service.APIConfig{CoAP: service.CoAPConfig{ID: "test", Config: net.Config{ExternalAddresses: []string{"localhost:12345"}}}},
		},
		{
			name:      "InvalidCoAPConfig",
			apiConfig: service.APIConfig{CoAP: service.CoAPConfig{Config: net.Config{ExternalAddresses: []string{"localhost:12345"}}}},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.apiConfig.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  service.Config
		wantErr bool
	}{
		{
			name:   "ValidConfig",
			config: service.Config{API: service.APIConfig{CoAP: service.CoAPConfig{ID: "test", Config: net.Config{ExternalAddresses: []string{"localhost:12345"}}}}},
		},
		{
			name:    "InvalidAPIConfig",
			config:  service.Config{API: service.APIConfig{CoAP: service.CoAPConfig{Config: net.Config{ExternalAddresses: []string{"localhost:12345"}}}}},
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
		})
	}
}
