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
	"fmt"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	cloudResource "github.com/plgd-dev/device/v2/bridge/resources/cloud"
	resourcesDevice "github.com/plgd-dev/device/v2/bridge/resources/device"
	"github.com/plgd-dev/device/v2/schema"
	cloudSchema "github.com/plgd-dev/device/v2/schema/cloud"
	plgdDevice "github.com/plgd-dev/device/v2/schema/device"
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
	cfg             Config
	resources       *sync.Map[string, Resource]
	cloudManager    *cloud.Manager
	onDeviceUpdated func(d *Device)
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

func (d *Device) ExportConfig() Config {
	cfg := d.cfg
	cfg.Cloud.Config = d.cloudManager.ExportConfig()
	return cfg
}

func (d *Device) GetProtocolIndependentID() uuid.UUID {
	return d.cfg.ProtocolIndependentID
}

type CloudConfig struct {
	Enabled bool
	cloud.Config
}

type Config struct {
	ID                    uuid.UUID
	Name                  string
	ProtocolIndependentID uuid.UUID
	ResourceTypes         []string
	MaxMessageSize        uint32
	Cloud                 CloudConfig
}

func (cfg *Config) Validate() error {
	if cfg.ProtocolIndependentID == uuid.Nil {
		return fmt.Errorf("protocolIndependentID is required")
	}
	if cfg.ID == uuid.Nil {
		cfg.ID = uuid.New()
	}

	if cfg.Name == "" {
		cfg.Name = "Unnamed"
	}
	return nil
}

func New(cfg Config, onDeviceUpdated func(d *Device), additionalProperties resourcesDevice.GetAdditionalPropertiesForResponseFunc) *Device {
	if onDeviceUpdated == nil {
		onDeviceUpdated = func(d *Device) {
			// do nothing
		}
	}
	cfg.ResourceTypes = resources.Unique(append(cfg.ResourceTypes, plgdDevice.ResourceType))
	d := &Device{
		cfg:             cfg,
		resources:       sync.NewMap[string, Resource](),
		onDeviceUpdated: onDeviceUpdated,
	}
	d.AddResource(resourcesDevice.New(plgdDevice.ResourceURI, d, additionalProperties))

	if cfg.Cloud.Enabled {
		d.cloudManager = cloud.New(d.cfg.ID, func() {
			d.onDeviceUpdated(d)
		}, d.HandleRequest, d.GetLinksFilteredBy, cfg.MaxMessageSize)
		d.AddResource(cloudResource.New(cloudSchema.ResourceURI, d.cloudManager))
		d.cloudManager.ImportConfig(cfg.Cloud.Config)
	}
	return d
}

func (d *Device) AddResource(resource Resource) {
	d.resources.Store(resource.GetHref(), resource)
}

func (d *Device) Init() {
	d.cloudManager.Init()
}

func (d *Device) UnregisterFromCloud() {
	if d.cloudManager != nil {
		d.cloudManager.Unregister()
	}
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
			Anchor:    "ocf://" + d.GetID().String(),
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
	r, ok := d.resources.LoadAndDelete(resourceHref)
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
	res, ok := d.resources.Load(req.URIPath())
	if !ok {
		return createResponseNotFound(req.Context(), uri, req.Token()), nil
	}
	return res.HandleRequest(req)
}

func (d *Device) Close() {
	d.cloudManager.Close()
	for _, resource := range d.resources.LoadAndDeleteAll() {
		resource.Close()
	}
}
