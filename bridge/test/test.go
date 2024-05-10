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
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	bridgeDeviceTD "github.com/plgd-dev/device/v2/bridge/device/thingDescription"
	thingDescriptionResource "github.com/plgd-dev/device/v2/bridge/resources/thingDescription"
	"github.com/plgd-dev/device/v2/bridge/service"
	"github.com/plgd-dev/device/v2/pkg/log"
	"github.com/plgd-dev/device/v2/schema"
	schemaCloud "github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/credential"
	schemaDevice "github.com/plgd-dev/device/v2/schema/device"
	schemaMaintenance "github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/device/v2/test"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/stretchr/testify/require"
	wotTD "github.com/web-of-things-open-source/thingdescription-go/thingDescription"
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

func NewBridgeService(t *testing.T, opts ...service.Option) *service.Service {
	opts = append([]service.Option{service.WithLogger(log.NewStdLogger(log.LevelDebug))}, opts...)
	s, err := service.New(MakeConfig(t), opts...)
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
	newDevice := func(di uuid.UUID, piid uuid.UUID) (service.Device, error) {
		cfg.ID = di
		cfg.ProtocolIndependentID = piid
		require.NoError(t, cfg.Validate())
		caPool := cloud.MakeCAPool(func() []*x509.Certificate {
			return test.GetRootCA(t)
		}, false)
		deviceOpts := []device.Option{
			device.WithCAPool(caPool),
			device.WithGetCertificates(func(string) []tls.Certificate {
				return []tls.Certificate{test.GetMfgCertificate(t)}
			}),
			device.WithLogger(device.NewLogger(di, log.LevelDebug)),
		}
		// allow to override default options
		deviceOpts = append(deviceOpts, opts...)
		return device.New(cfg, deviceOpts...)
	}
	d, err := s.CreateDevice(cfg.ID, newDevice)
	require.NoError(t, err)
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

func GetPropertyElement(td wotTD.ThingDescription, device bridgeDeviceTD.Device, endpoint string, resourceHref string, resource bridgeDeviceTD.Resource, contentType message.MediaType) (wotTD.PropertyElement, bool) {
	propElement, ok := td.Properties[resourceHref]
	if !ok {
		return wotTD.PropertyElement{}, false
	}
	var f bridgeDeviceTD.CreateFormsFunc
	if endpoint != "" {
		f = bridgeDeviceTD.CreateCOAPForms
	}
	propElement, err := bridgeDeviceTD.PatchPropertyElement(propElement, resource.GetResourceTypes(), device.GetID(), resource.GetHref(),
		resource.SupportsOperations(), contentType, f)
	return propElement, err == nil
}

func NewBridgedDevice(t *testing.T, s *service.Service, id string, cloudEnabled, credentialEnabled, thingDescriptionEnabled bool, opts ...device.Option) service.Device {
	deviceID, err := uuid.Parse(id)
	require.NoError(t, err)
	cfg := makeDeviceConfig(deviceID, cloudEnabled, credentialEnabled)
	if thingDescriptionEnabled {
		td, err := ThingDescription(deviceID, "", cloudEnabled, credentialEnabled)
		require.NoError(t, err)
		return NewBridgedDeviceWithThingDescription(t, s, id, cloudEnabled, credentialEnabled, &td, opts...)
	}
	return NewBridgedDeviceWithConfig(t, s, cfg, opts...)
}

func NewBridgedDeviceWithThingDescription(t *testing.T, s *service.Service, id string, cloudEnabled, credentialEnabled bool, td *wotTD.ThingDescription, opts ...device.Option) service.Device {
	u, err := uuid.Parse(id)
	require.NoError(t, err)
	cfg := makeDeviceConfig(u, cloudEnabled, credentialEnabled)
	if td != nil {
		opts = append(opts, device.WithThingDescription(func(_ context.Context, device *device.Device, endpoints schema.Endpoints) *wotTD.ThingDescription {
			endpoint := ""
			if len(endpoints) > 0 {
				endpoint = endpoints[0].URI
			}
			newTD := bridgeDeviceTD.PatchThingDescription(*td, device, endpoint,
				func(resourceHref string, resource bridgeDeviceTD.Resource) (wotTD.PropertyElement, bool) {
					return GetPropertyElement(*td, device, endpoint, resourceHref, resource, message.AppCBOR)
				})
			return &newTD
		}))
	}
	return NewBridgedDeviceWithConfig(t, s, cfg, opts...)
}

func getOCFResourcesProperties(deviceID uuid.UUID, baseURL string, cloudEnabled, credentialEnabled bool) (map[string]wotTD.PropertyElement, error) {
	properties := make(map[string]wotTD.PropertyElement)
	deviceResource, ok := thingDescriptionResource.GetOCFResourcePropertyElement(schemaDevice.ResourceURI)
	if !ok {
		return nil, errors.New("device resource not found")
	}
	deviceResource, err := thingDescriptionResource.PatchDeviceResourcePropertyElement(deviceResource, deviceID, baseURL, message.AppCBOR, "", bridgeDeviceTD.CreateCOAPForms)
	if err != nil {
		return nil, err
	}
	properties[schemaDevice.ResourceURI] = deviceResource

	maintenanceResource, ok := thingDescriptionResource.GetOCFResourcePropertyElement(schemaMaintenance.ResourceURI)
	if !ok {
		return nil, errors.New("maintenance resource not found")
	}
	properties[schemaMaintenance.ResourceURI] = maintenanceResource
	maintenanceResource, err = thingDescriptionResource.PatchMaintenanceResourcePropertyElement(maintenanceResource, deviceID, baseURL, message.AppCBOR, bridgeDeviceTD.CreateCOAPForms)
	if err != nil {
		return nil, err
	}
	properties[schemaMaintenance.ResourceURI] = maintenanceResource

	if cloudEnabled {
		cloudResource, ok := thingDescriptionResource.GetOCFResourcePropertyElement(schemaCloud.ResourceURI)
		if !ok {
			return nil, errors.New("cloud resource not found")
		}
		cloudResource, err = thingDescriptionResource.PatchCloudResourcePropertyElement(cloudResource, deviceID, baseURL, message.AppCBOR, bridgeDeviceTD.CreateCOAPForms)
		if err != nil {
			return nil, err
		}
		properties[schemaCloud.ResourceURI] = cloudResource
	}

	if credentialEnabled {
		credentialResource, ok := thingDescriptionResource.GetOCFResourcePropertyElement(credential.ResourceURI)
		if !ok {
			return nil, errors.New("credential resource not found")
		}
		credentialResource, err = thingDescriptionResource.PatchCredentialResourcePropertyElement(credentialResource, deviceID, baseURL, message.AppCBOR, bridgeDeviceTD.CreateCOAPForms)
		if err != nil {
			return nil, err
		}
		properties[credential.ResourceURI] = credentialResource
	}
	return properties, nil
}

func ThingDescription(deviceID uuid.UUID, baseURL string, cloudEnabled, credentialEnabled bool) (wotTD.ThingDescription, error) {
	td := wotTD.ThingDescription{}
	td.Context = &bridgeDeviceTD.Context
	td.Type = &wotTD.TypeDeclaration{StringArray: []string{"Thing"}}
	id, err := bridgeDeviceTD.GetThingDescriptionID(deviceID.String())
	if err != nil {
		return wotTD.ThingDescription{}, err
	}
	td.ID = id
	if baseURL != "" {
		base, errP := url.Parse(baseURL)
		if errP != nil {
			return wotTD.ThingDescription{}, errP
		}
		td.Base = *base
	}

	properties, err := getOCFResourcesProperties(deviceID, baseURL, cloudEnabled, credentialEnabled)
	if err != nil {
		return wotTD.ThingDescription{}, err
	}
	td.Properties = properties
	return td, nil
}
