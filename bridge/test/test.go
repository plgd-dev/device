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
	"testing"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/device/thingDescription"
	"github.com/plgd-dev/device/v2/bridge/service"
	"github.com/plgd-dev/device/v2/pkg/log"
	"github.com/plgd-dev/device/v2/schema"
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
		opts = append(opts, device.WithThingDescription(func(_ context.Context, device *device.Device, endpoints schema.Endpoints) *wotTD.ThingDescription {
			endpoint := ""
			if len(endpoints) > 0 {
				endpoint = endpoints[0].URI
			}
			newTD := thingDescription.PatchThingDescription(td, device, endpoint,
				func(resourceHref string, resource thingDescription.Resource) (wotTD.PropertyElement, bool) {
					return GetPropertyElement(td, device, endpoint, resourceHref, resource)
				})
			return &newTD
		}))
	}
	return NewBridgedDeviceWithConfig(t, s, cfg, opts...)
}

func ThingDescription(cloudEnabled, credentialEnabled bool) (wotTD.ThingDescription, error) {
	tdJson := `{
		"@context": "https://www.w3.org/2019/wot/td/v1",
		"@type": [
			"Thing"
		],
		"id": "urn:uuid:bridge",
		"properties": {
			"/oic/d": {
				"title": "Device Information",
				"type": "object",
				"properties": {
					"piid": {
						"title": "Protocol Interface ID",
						"type": "string",
						"readOnly": true,
						"format": "uuid"
					},
					"n": {
						"title": "Device Name",
						"type": "string",
						"readOnly": true
					},
					"di": {
						"title": "Device ID",
						"type": "string",
						"readOnly": true,
						"format": "uuid"
					}
				}
			},
			"/oic/mnt": {
				"title": "Maintenance",
				"type": "object",
				"properties": {
					"fr": {
						"title": "Factory Reset",
						"type": "boolean"
					}
				}
			}`
	if cloudEnabled {
		tdJson += `,
			"/CoapCloudConfResURI": {
					"title": "CoapCloudConfResURI",
					"type": "object",
					"properties": {
						"apn": {
							"title": "Authorization provider name",
							"type": "string"
						},
						"cis": {
							"title": "Cloud interface server",
							"type": "string",
							"format": "uri"
						},
						"sid": {
							"title": "Cloud ID",
							"type": "string",
							"format": "uuid"
						},
						"at": {
							"title": "Access token",
							"type": "string"
						},
						"cps": {
							"title": "Provisioning status",
							"type": "string",
							"enum": [
								"uninitialized",
								"readytoregister",
								"registering",
								"registered",
								"failed"
							]
						},
						"clec": {
							"title": "Last error code",
							"type": "integer"
						}
					}
				}`
	}

	if credentialEnabled {
		tdJson += `,
			"/oic/sec/cred": {
				"title": "Credential",
				"type": "object",
				"properties": {
					"credid": {
						"title": "Credential ID",
						"type": "integer"
					},
					"credtype": {
						"title": "Credential Type",
						"type": "integer",
						"enum": [
							0,
							1,
							2,
							4,
							8,
							16,
							32
						]
					},
					"subjectuuid": {
						"title": "Subject UUID",
						"type": "string"
					},
					"credusage": {
						"title": "Credential Usage",
						"type": "string",
						"enum": [
							"oic.sec.cred.trustca",
							"oic.sec.cred.cert",
							"oic.sec.cred.rolecert",
							"oic.sec.cred.mfgtrustca",
							"oic.sec.cred.mfgcert"
						]
					},
					"privatedata": {
						"title": "Private Data",
						"type": "object",
						"properties": {
							"data": {
								"title": "Data",
								"type": "string"
							},
							"encoding": {
								"title": "Encoding",
								"type": "string",
								"enum": [
									"oic.sec.encoding.jwt",
									"oic.sec.encoding.cwt",
									"oic.sec.encoding.base64",
									"oic.sec.encoding.uri",
									"oic.sec.encoding.handle",
									"oic.sec.encoding.raw"
								]
							}
						}
					},
					"publicdata": {
						"title": "Public Data",
						"type": "object",
						"properties": {
							"data": {
								"title": "Data",
								"type": "string"
							},
							"encoding": {
								"title": "Encoding",
								"type": "string",
								"enum": [
									"oic.sec.encoding.jwt",
									"oic.sec.encoding.cwt",
									"oic.sec.encoding.base64",
									"oic.sec.encoding.uri",
									"oic.sec.encoding.pem",
									"oic.sec.encoding.der",
									"oic.sec.encoding.raw"
								]
							}
						}
					},
					"roleid": {
						"title": "Role ID",
						"type": "object",
						"properties": {
							"authority": {
								"title": "Authority",
								"type": "string"
							},
							"role": {
								"title": "Role",
								"type": "string"
							}
						}
					},
					"tag": {
						"title": "Tag",
						"type": "string"
					}
				}
			}`
	}

	tdJson += `
		}
	}`

	return wotTD.UnmarshalThingDescription([]byte(tdJson))
}
