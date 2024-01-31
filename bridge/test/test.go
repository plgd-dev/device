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
	"crypto/tls"
	"crypto/x509"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/service"
	"github.com/plgd-dev/device/v2/test"
	"github.com/stretchr/testify/require"
)

const (
	BRIDGE_SERVICE_PIID     = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	BRIDGE_DEVICE_HOST      = "127.0.0.1:0" // 0 means random port
	BRIDGE_DEVICE_HOST_IPv6 = "[::1]:0"     // 0 means random port
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

func RunBridgeService(s *service.Service) func() error {
	cleanup := func() error {
		return s.Shutdown()
	}
	go func() {
		_ = s.Serve()
	}()
	return cleanup
}

func NewBridgedDeviceWithConfig(t *testing.T, s *service.Service, cfg device.Config, opts ...device.Option) service.Device {
	newDevice := func(di uuid.UUID, piid uuid.UUID) service.Device {
		cfg.ID = di
		cfg.ProtocolIndependentID = piid
		require.NoError(t, cfg.Validate())
		caPool := cloud.MakeCAPool(func() []*x509.Certificate {
			return test.GetRootCA(t)
		}, false)
		deviceOpts := []device.Option{device.WithCAPool(caPool), device.WithGetCertificates(func(deviceID string) []tls.Certificate {
			return []tls.Certificate{test.GetMfgCertificate(t)}
		})}
		// allow to override default options
		deviceOpts = append(deviceOpts, opts...)
		dev, err := device.New(cfg, deviceOpts...)
		require.NoError(t, err)
		return dev
	}
	d, ok := s.CreateDevice(cfg.ID, newDevice)
	require.True(t, ok)
	d.Init()
	return d
}

func makeDeviceConfig(id uuid.UUID, cloudEnabled bool, credentialEnabled bool) device.Config {
	cfg := device.Config{
		ID:             id,
		Name:           "bridged-device",
		ResourceTypes:  []string{"oic.d.virtual"},
		MaxMessageSize: 1024 * 256,
	}
	cfg.Cloud.Enabled = cloudEnabled
	if cloudEnabled {
		cfg.Cloud.CloudID = test.CloudSID()
	}
	cfg.Credential.Enabled = credentialEnabled
	return cfg
}

func NewBridgedDevice(t *testing.T, s *service.Service, id string, cloudEnabled bool, credentialEnabled bool, opts ...device.Option) service.Device {
	u, err := uuid.Parse(id)
	require.NoError(t, err)
	cfg := makeDeviceConfig(u, cloudEnabled, credentialEnabled)
	return NewBridgedDeviceWithConfig(t, s, cfg, opts...)
}
