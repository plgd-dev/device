/****************************************************************************
 *
 * Copyright (c) 2023 plgd.dev s.r.o.
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

package device

import (
	"bytes"
	"context"
	"crypto/x509"
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/device/credential"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	cloudResource "github.com/plgd-dev/device/v2/bridge/resources/cloud"
	resourcesDevice "github.com/plgd-dev/device/v2/bridge/resources/device"
	"github.com/plgd-dev/device/v2/bridge/resources/discovery"
	"github.com/plgd-dev/device/v2/bridge/resources/maintenance"
	credentialResource "github.com/plgd-dev/device/v2/bridge/resources/secure/credential"
	"github.com/plgd-dev/device/v2/schema"
	cloudSchema "github.com/plgd-dev/device/v2/schema/cloud"
	credentialSchema "github.com/plgd-dev/device/v2/schema/credential"
	plgdDevice "github.com/plgd-dev/device/v2/schema/device"
	maintenanceSchema "github.com/plgd-dev/device/v2/schema/maintenance"
	plgdResources "github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/pkg/sync"
)

type Resource interface {
	Close()
	ETag() []byte
	GetHref() string
	GetResourceTypes() []string
	GetResourceInterfaces() []string
	HandleRequest(req *net.Request) (*pool.Message, error)
	GetPolicyBitMask() schema.BitMask
	SetObserveHandler(createSubscription resources.CreateSubscriptionFunc)
	UpdateETag()
}

type Device struct {
	cfg               Config
	resources         *sync.Map[string, Resource]
	cloudManager      *cloud.Manager
	credentialManager *credential.Manager
	onDeviceUpdated   func(d *Device)
}

func (d *Device) GetID() uuid.UUID {
	return d.cfg.ID
}

func (d *Device) GetName() string {
	return d.cfg.Name
}

func (d *Device) GetResourceTypes() []string {
	return d.cfg.ResourceTypes
}

func (d *Device) GetProtocolIndependentID() uuid.UUID {
	return d.cfg.ProtocolIndependentID
}

func (d *Device) ExportConfig() Config {
	cfg := d.cfg
	if d.cloudManager != nil {
		cfg.Cloud.Config = d.cloudManager.ExportConfig()
	} else {
		cfg.Cloud.Enabled = false
	}
	if d.credentialManager != nil {
		cfg.Credential.Config = d.credentialManager.ExportConfig()
	} else {
		cfg.Credential.Enabled = false
	}
	return cfg
}

type OnDeviceUpdated func(d *Device)

type CAPoolGetter interface {
	IsValid() bool
	GetPool() (*x509.CertPool, error)
}

type OptionsCfg struct {
	onDeviceUpdated         OnDeviceUpdated
	getAdditionalProperties resourcesDevice.GetAdditionalPropertiesForResponseFunc
	getCertificates         cloud.GetCertificates
	caPool                  CAPoolGetter
}

type Option func(*OptionsCfg)

func WithOnDeviceUpdated(onDeviceUpdated OnDeviceUpdated) Option {
	return func(o *OptionsCfg) {
		o.onDeviceUpdated = onDeviceUpdated
	}
}

func WithGetAdditionalPropertiesForResponse(getAdditionalProperties resourcesDevice.GetAdditionalPropertiesForResponseFunc) Option {
	return func(o *OptionsCfg) {
		o.getAdditionalProperties = getAdditionalProperties
	}
}

func WithGetCertificates(getCertificates cloud.GetCertificates) Option {
	return func(o *OptionsCfg) {
		o.getCertificates = getCertificates
	}
}

func WithCAPool(caPool CAPoolGetter) Option {
	return func(o *OptionsCfg) {
		o.caPool = caPool
	}
}

func New(cfg Config, opts ...Option) (*Device, error) {
	o := OptionsCfg{
		onDeviceUpdated: func(d *Device) {
			// do nothing
		},
		getAdditionalProperties: func() map[string]interface{} { return nil },
		caPool:                  cloud.MakeCAPool(nil, false),
	}
	for _, opt := range opts {
		opt(&o)
	}

	cfg.ResourceTypes = resources.Unique(append(cfg.ResourceTypes, plgdDevice.ResourceType))
	d := &Device{
		cfg:             cfg,
		resources:       sync.NewMap[string, Resource](),
		onDeviceUpdated: o.onDeviceUpdated,
	}

	cloudOpts := []cloud.Option{cloud.WithMaxMessageSize(cfg.MaxMessageSize)}
	if cfg.Credential.Enabled {
		d.credentialManager = credential.New(func() {
			d.onDeviceUpdated(d)
		})
		d.AddResource(credentialResource.New(credentialSchema.ResourceURI, d.credentialManager))
		o.caPool = credential.MakeCAPool(o.caPool, d.credentialManager.GetCAPool)
		cloudOpts = append(cloudOpts, cloud.WithRemoveCloudCAs(d.credentialManager.RemoveCredentialsBySubjects))
	}
	if cfg.Cloud.Enabled {
		if o.getCertificates != nil {
			cloudOpts = append(cloudOpts, cloud.WithGetCertificates(o.getCertificates))
		}
		cm, err := cloud.New(d.cfg.ID, func() {
			d.onDeviceUpdated(d)
		}, d.HandleRequest, d.GetLinksFilteredBy, o.caPool, cloudOpts...)
		if err != nil {
			return nil, fmt.Errorf("cannot create cloud manager: %w", err)
		}
		d.cloudManager = cm
		d.AddResource(cloudResource.New(cloudSchema.ResourceURI, d.cloudManager))
		d.cloudManager.ImportConfig(cfg.Cloud.Config)
	}

	d.AddResource(resourcesDevice.New(plgdDevice.ResourceURI, d, o.getAdditionalProperties))
	// oic/res is not discoverable
	discoverResource := discovery.New(plgdResources.ResourceURI, d.GetLinks)
	discoverResource.PolicyBitMask = schema.Discoverable
	d.AddResource(discoverResource)

	d.AddResource(maintenance.New(maintenanceSchema.ResourceURI, func() {
		if d.cloudManager != nil {
			d.cloudManager.Unregister()
		}
	}))

	return d, nil
}

func (d *Device) AddResource(resource Resource) {
	d.resources.Store(resource.GetHref(), resource)
}

func (d *Device) Init() {
	if d.cloudManager != nil {
		d.cloudManager.Init()
	}
}

func (d *Device) GetCloudManager() *cloud.Manager {
	return d.cloudManager
}

func (d *Device) Range(f func(resourceHref string, resource Resource) bool) {
	d.resources.Range(f)
}

func (d *Device) GetResource(resourceHref string) (Resource, bool) {
	return d.resources.Load(resourceHref)
}

func hasResourceTypes(resourceTypes []string, oneOf []string) bool {
	for _, rt := range oneOf {
		for _, rrt := range resourceTypes {
			if rt == rrt {
				return true
			}
		}
	}
	return false
}

func (d *Device) GetLinksFilteredBy(endpoints schema.Endpoints, deviceIDfilter uuid.UUID, resourceTypesFitler []string, policyBitMaskFitler schema.BitMask) (links schema.ResourceLinks) {
	di := deviceIDfilter
	if di != uuid.Nil && di != d.GetID() {
		return nil
	}
	links = make(schema.ResourceLinks, 0, d.resources.Length())
	d.resources.Range(func(key string, resource Resource) bool {
		if len(resourceTypesFitler) > 0 && !hasResourceTypes(resource.GetResourceTypes(), resourceTypesFitler) {
			return true
		}
		if policyBitMaskFitler != 0 && resource.GetPolicyBitMask()&policyBitMaskFitler == 0 {
			return true
		}
		links = append(links, schema.ResourceLink{
			Href:          key,
			ResourceTypes: resource.GetResourceTypes(),
			Interfaces:    resource.GetResourceInterfaces(),
			Policy: &schema.Policy{
				BitMask: resource.GetPolicyBitMask() & (^resources.PublishToCloud),
			},
			DeviceID:  d.GetID().String(),
			Endpoints: endpoints,
		})
		return true
	})
	return links
}

func (d *Device) GetLinks(request *net.Request) (links schema.ResourceLinks) {
	return d.GetLinksFilteredBy(request.Endpoints, request.DeviceID(), request.ResourceTypes(), 0)
}

func (d *Device) LoadAndDeleteResource(resourceHref string) (Resource, bool) {
	return d.resources.LoadAndDelete(resourceHref)
}

func (d *Device) CloseAndDeleteResource(resourceHref string) bool {
	r, ok := d.LoadAndDeleteResource(resourceHref)
	if ok {
		r.Close()
	}
	return ok
}

func createResponseNotFound(ctx context.Context, uri string, token message.Token) *pool.Message {
	msg := pool.NewMessage(ctx)
	msg.SetCode(codes.NotFound)
	msg.SetToken(token)
	msg.SetBody(bytes.NewReader([]byte(fmt.Sprintf("uri %v not found", uri))))
	return msg
}

func (d *Device) HandleRequest(req *net.Request) (*pool.Message, error) {
	uri := req.URIPath()
	res, ok := d.resources.Load(uri)
	if !ok {
		return createResponseNotFound(req.Context(), uri, req.Token()), nil
	}
	return res.HandleRequest(req)
}

func (d *Device) Close() {
	if d.cloudManager != nil {
		d.cloudManager.Close()
	}
	if d.credentialManager != nil {
		d.credentialManager.Close()
	}
	for _, resource := range d.resources.LoadAndDeleteAll() {
		resource.Close()
	}
}
