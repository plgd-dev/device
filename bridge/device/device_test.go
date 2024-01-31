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
	"time"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/resources"
	cloudSchema "github.com/plgd-dev/device/v2/schema/cloud"
	plgdDevice "github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	maintenanceSchema "github.com/plgd-dev/device/v2/schema/maintenance"
	plgdResources "github.com/plgd-dev/device/v2/schema/resources"
	"github.com/stretchr/testify/require"
)

var (
	deviceCfg = device.Config{
		ID:                    uuid.New(),
		Name:                  "test",
		ProtocolIndependentID: uuid.New(),
		ResourceTypes:         []string{"rt1", "rt2"},
		MaxMessageSize:        1024,
	}

	cloudCfg = device.CloudConfig{
		Enabled: true,
		Config: cloud.Config{
			AccessToken:           "access token",
			UserID:                "user id",
			RefreshToken:          "refresh token",
			ValidUntil:            time.Date(2042, 4, 2, 0, 0, 0, 0, time.UTC), // 4.2.2042
			AuthorizationProvider: "auth provider",
			CloudID:               "cloud id",
			CloudURL:              "cloud url",
		},
	}
)

func TestNewDevice(t *testing.T) {
	cfg := deviceCfg
	dev, err := device.New(cfg)
	require.NoError(t, err)
	require.Equal(t, cfg.ID, dev.GetID())
	require.Equal(t, cfg.Name, dev.GetName())
	require.Equal(t, cfg.ProtocolIndependentID, dev.GetProtocolIndependentID())
	// oic.wk.d will be added automatically
	require.Subset(t, dev.GetResourceTypes(), cfg.ResourceTypes)
	require.Contains(t, dev.GetResourceTypes(), "oic.wk.d")

	cfg.ResourceTypes = append(cfg.ResourceTypes, "oic.wk.d")
	require.Equal(t, cfg, dev.ExportConfig())
}

func TestNewDeviceWithCloud(t *testing.T) {
	cfg := deviceCfg
	cfg.Cloud = cloudCfg

	dev, err := device.New(cfg, device.WithCAPool(cloud.MakeCAPool(nil, true)))
	require.NoError(t, err)
	cfg.ResourceTypes = append(cfg.ResourceTypes, "oic.wk.d")
	require.Equal(t, cfg, dev.ExportConfig())
}

func TestGetResource(t *testing.T) {
	dev, err := device.New(deviceCfg)
	require.NoError(t, err)
	_, ok := dev.GetResource(plgdDevice.ResourceURI)
	require.True(t, ok)

	// cloud was not enabled in the cfg
	_, ok = dev.GetResource(cloudSchema.ResourceURI)
	require.False(t, ok)

	// not existing resource
	_, ok = dev.GetResource("/no-resource")
	require.False(t, ok)
}

func TestRangeResources(t *testing.T) {
	cfg := deviceCfg
	dev, err := device.New(cfg)
	require.NoError(t, err)
	resourceHrefs := []string{}
	dev.Range(func(href string, _ device.Resource) bool {
		resourceHrefs = append(resourceHrefs, href)
		return true
	})
	// default resources: device, discovery and maintenance
	require.Len(t, resourceHrefs, 3)
	require.Contains(t, resourceHrefs, plgdDevice.ResourceURI)
	require.Contains(t, resourceHrefs, plgdResources.ResourceURI)
	require.Contains(t, resourceHrefs, maintenanceSchema.ResourceURI)

	cfg.Cloud = cloudCfg
	devWithCloud, err := device.New(cfg, device.WithCAPool(cloud.MakeCAPool(nil, true)))
	require.NoError(t, err)
	resourceHrefs = []string{}
	devWithCloud.Range(func(href string, _ device.Resource) bool {
		resourceHrefs = append(resourceHrefs, href)
		return true
	})
	// default resources: device, discovery, maintenance and cloud
	require.Len(t, resourceHrefs, 4)
	require.Contains(t, resourceHrefs, plgdDevice.ResourceURI)
	require.Contains(t, resourceHrefs, plgdResources.ResourceURI)
	require.Contains(t, resourceHrefs, maintenanceSchema.ResourceURI)
	require.Contains(t, resourceHrefs, cloudSchema.ResourceURI)
}

func TestLoadAndDeleteResource(t *testing.T) {
	dev, err := device.New(deviceCfg)
	require.NoError(t, err)

	_, ok := dev.LoadAndDeleteResource("/fail")
	require.False(t, ok)

	res := resources.NewResource("/test", nil, nil, []string{"oic.d.virtual", "oic.d.test"}, []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_RW})
	dev.AddResource(res)
	_, ok = dev.GetResource(res.GetHref())
	require.True(t, ok)

	_, ok = dev.LoadAndDeleteResource(res.GetHref())
	require.True(t, ok)
	_, ok = dev.GetResource(res.GetHref())
	require.False(t, ok)
}

func TestCloseAndDeleteResource(t *testing.T) {
	dev, err := device.New(deviceCfg)
	require.NoError(t, err)

	ok := dev.CloseAndDeleteResource("/fail")
	require.False(t, ok)

	res := resources.NewResource("/test", nil, nil, []string{"oic.d.virtual", "oic.d.test"}, []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_RW})
	dev.AddResource(res)
	_, ok = dev.GetResource(res.GetHref())
	require.True(t, ok)

	ok = dev.CloseAndDeleteResource(res.GetHref())
	require.True(t, ok)
	_, ok = dev.GetResource(res.GetHref())
	require.False(t, ok)
}
