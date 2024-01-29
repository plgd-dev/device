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

package test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/service"
	"github.com/stretchr/testify/require"
)

const (
	BRIDGE_SERVICE_PIID     = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	BRIDGE_DEVICE_HOST      = "127.0.0.1:15000"
	BRIDGE_DEVICE_HOST_IPv6 = "[::1]:15001"
)

func MakeConfig(t *testing.T) service.Config {
	var cfg service.Config
	cfg.API.CoAP.ID = BRIDGE_SERVICE_PIID
	cfg.API.CoAP.Config.ExternalAddresses = []string{BRIDGE_DEVICE_HOST, BRIDGE_DEVICE_HOST_IPv6}
	require.NoError(t, cfg.API.Validate())
	return cfg
}

func NewBridgeService(t *testing.T) *service.Service {
	s, err := service.New(MakeConfig(t))
	require.NoError(t, err)
	return s
}

func RunBridgeService(s *service.Service) func() {
	cleanup := func() {
		_ = s.Shutdown()
	}
	go func() {
		_ = s.Serve()
	}()
	return cleanup
}

func NewBridgedDevice(t *testing.T, s *service.Service, cloudEnabled bool, id string) service.Device {
	newDevice := func(id uuid.UUID, piid uuid.UUID) service.Device {
		cfg := device.Config{
			Name:                  "bridged-device",
			ResourceTypes:         []string{"oic.d.virtual"},
			ID:                    id,
			ProtocolIndependentID: piid,
			MaxMessageSize:        1024 * 256,
		}
		if cloudEnabled {
			cfg.Cloud.Enabled = true
		}
		require.NoError(t, cfg.Validate())
		return device.New(cfg, nil, nil)
	}
	u, err := uuid.Parse(id)
	require.NoError(t, err)
	d, ok := s.CreateDevice(u, newDevice)
	require.True(t, ok)
	d.Init()
	return d
}
