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
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/device/thingDescription"
	"github.com/plgd-dev/device/v2/bridge/service"
	"github.com/plgd-dev/device/v2/pkg/log"
	"github.com/plgd-dev/device/v2/schema"
	schemaCloud "github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/test"
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

func GetPropertyElement(td wotTD.ThingDescription, device thingDescription.Device, endpoint string, resourceHref string, resource thingDescription.Resource) (wotTD.PropertyElement, bool) {
	propElement, ok := td.Properties[resourceHref]
	if !ok {
		return wotTD.PropertyElement{}, false
	}
	propElement = thingDescription.PatchPropertyElement(propElement, device.GetID(), resource, endpoint != "")
	return propElement, true
}

func NewBridgedDevice(t *testing.T, s *service.Service, id string, cloudEnabled, credentialEnabled, thingDescriptionEnabled bool, opts ...device.Option) service.Device {
	u, err := uuid.Parse(id)
	require.NoError(t, err)
	cfg := makeDeviceConfig(u, cloudEnabled, credentialEnabled)
	if thingDescriptionEnabled {
		td, err := ThingDescription(cloudEnabled, credentialEnabled)
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
			newTD := thingDescription.PatchThingDescription(*td, device, endpoint,
				func(resourceHref string, resource thingDescription.Resource) (wotTD.PropertyElement, bool) {
					return GetPropertyElement(*td, device, endpoint, resourceHref, resource)
				})
			return &newTD
		}))
	}
	return NewBridgedDeviceWithConfig(t, s, cfg, opts...)
}

func ThingDescription(cloudEnabled, credentialEnabled bool) (wotTD.ThingDescription, error) {
	type ThingDescription struct {
		Context    string                 `json:"@context"`
		Type       []string               `json:"@type"`
		ID         string                 `json:"id"`
		Properties map[string]interface{} `json:"properties"`
	}

	td := ThingDescription{
		Context: "https://www.w3.org/2019/wot/td/v1",
		Type:    []string{"Thing"},
		ID:      "urn:uuid:bridge",
		Properties: map[string]interface{}{
			"/oic/d": map[string]interface{}{
				"title": "Device Information",
				"type":  "object",
				"properties": map[string]interface{}{
					"piid": map[string]interface{}{
						"title":    "Protocol Interface ID",
						"type":     "string",
						"readOnly": true,
						"format":   "uuid",
					},
					"n": map[string]interface{}{
						"title":    "Device Name",
						"type":     "string",
						"readOnly": true,
					},
					"di": map[string]interface{}{
						"title":    "Device ID",
						"type":     "string",
						"readOnly": true,
						"format":   "uuid",
					},
				},
			},
			"/oic/mnt": map[string]interface{}{
				"title": "Maintenance",
				"type":  "object",
				"properties": map[string]interface{}{
					"fr": map[string]interface{}{
						"title": "Factory Reset",
						"type":  "boolean",
					},
				},
			},
		},
	}

	if cloudEnabled {
		td.Properties[schemaCloud.ResourceURI] = map[string]interface{}{
			"title": "CoapCloudConfResURI",
			"type":  "object",
			"properties": map[string]interface{}{
				"apn": map[string]interface{}{
					"title": "Authorization provider name",
					"type":  "string",
				},
				"cis": map[string]interface{}{
					"title":  "Cloud interface server",
					"type":   "string",
					"format": "uri",
				},
				"sid": map[string]interface{}{
					"title":  "Cloud ID",
					"type":   "string",
					"format": "uuid",
				},
				"at": map[string]interface{}{
					"title": "Access token",
					"type":  "string",
				},
				"cps": map[string]interface{}{
					"title": "Provisioning status",
					"type":  "string",
					"enum": []schemaCloud.ProvisioningStatus{
						schemaCloud.ProvisioningStatus_UNINITIALIZED,
						schemaCloud.ProvisioningStatus_READY_TO_REGISTER,
						schemaCloud.ProvisioningStatus_REGISTERING,
						schemaCloud.ProvisioningStatus_REGISTERED,
						schemaCloud.ProvisioningStatus_FAILED,
					},
				},
				"clec": map[string]interface{}{
					"title": "Last error code",
					"type":  "integer",
				},
			},
		}
	}

	if credentialEnabled {
		td.Properties[credential.ResourceURI] = map[string]interface{}{
			"title": "Credential",
			"type":  "object",
			"properties": map[string]interface{}{
				"credid": map[string]interface{}{
					"title": "Credential ID",
					"type":  "integer",
				},
				"credtype": map[string]interface{}{
					"title": "Credential Type",
					"type":  "integer",
					"enum": []int{
						int(credential.CredentialType_EMPTY),
						int(credential.CredentialType_SYMMETRIC_PAIR_WISE),
						int(credential.CredentialType_SYMMETRIC_GROUP),
						int(credential.CredentialType_ASYMMETRIC_SIGNING),
						int(credential.CredentialType_ASYMMETRIC_SIGNING_WITH_CERTIFICATE),
						int(credential.CredentialType_PIN_OR_PASSWORD),
						int(credential.CredentialType_ASYMMETRIC_ENCRYPTION_KEY),
					},
				},
				"subjectuuid": map[string]interface{}{
					"title": "Subject UUID",
					"type":  "string",
				},
				"credusage": map[string]interface{}{
					"title": "Credential Usage",
					"type":  "string",
					"enum": []credential.CredentialUsage{
						credential.CredentialUsage_TRUST_CA,
						credential.CredentialUsage_CERT,
						credential.CredentialUsage_ROLE_CERT,
						credential.CredentialUsage_MFG_TRUST_CA,
						credential.CredentialUsage_MFG_CERT,
					},
				},
				"privatedata": map[string]interface{}{
					"title": "Private Data",
					"type":  "object",
					"properties": map[string]interface{}{
						"data": map[string]interface{}{
							"title": "Data",
							"type":  "string",
						},
						"encoding": map[string]interface{}{
							"title": "Encoding",
							"type":  "string",
							"enum": []credential.CredentialPrivateDataEncoding{
								credential.CredentialPrivateDataEncoding_JWT,
								credential.CredentialPrivateDataEncoding_CWT,
								credential.CredentialPrivateDataEncoding_BASE64,
								credential.CredentialPrivateDataEncoding_URI,
								credential.CredentialPrivateDataEncoding_HANDLE,
								credential.CredentialPrivateDataEncoding_RAW,
							},
						},
					},
				},
				"publicdata": map[string]interface{}{
					"title": "Public Data",
					"type":  "object",
					"properties": map[string]interface{}{
						"data": map[string]interface{}{
							"title": "Data",
							"type":  "string",
						},
						"encoding": map[string]interface{}{
							"title": "Encoding",
							"type":  "string",
							"enum": []credential.CredentialPublicDataEncoding{
								credential.CredentialPublicDataEncoding_JWT,
								credential.CredentialPublicDataEncoding_CWT,
								credential.CredentialPublicDataEncoding_BASE64,
								credential.CredentialPublicDataEncoding_URI,
								credential.CredentialPublicDataEncoding_PEM,
								credential.CredentialPublicDataEncoding_DER,
								credential.CredentialPublicDataEncoding_RAW,
							},
						},
					},
				},
				"roleid": map[string]interface{}{
					"title": "Role ID",
					"type":  "object",
					"properties": map[string]interface{}{
						"authority": map[string]interface{}{
							"title": "Authority",
							"type":  "string",
						},
						"role": map[string]interface{}{
							"title": "Role",
							"type":  "string",
						},
					},
				},
				"tag": map[string]interface{}{
					"title": "Tag",
					"type":  "string",
				},
			},
		}
	}

	tdJson, err := json.Marshal(td)
	if err != nil {
		return wotTD.ThingDescription{}, err
	}
	return wotTD.UnmarshalThingDescription(tdJson)
}
