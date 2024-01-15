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
	"io"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	cloudResource "github.com/plgd-dev/device/v2/bridge/resources/cloud"
	"github.com/plgd-dev/device/v2/schema"
	plgdDevice "github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/pkg/sync"
	"gopkg.in/yaml.v3"
)

type Device struct {
	cfg             Config                                 `yaml:",inline"`
	resources       *sync.Map[string, *resources.Resource] `yaml:"-"`
	cloudManager    *cloud.Manager                         `yaml:"cloudManager"`
	onDeviceUpdated func(d *Device)
	data            any
}

func (d *Device) GetID() string {
	return resources.ToUUID(d.cfg.ID).String()
}

func (d *Device) GetRawID() string {
	return d.cfg.ID
}

func (d *Device) GetName() string {
	return d.cfg.Name
}

func (d *Device) GetResourceTypes() []string {
	return d.cfg.ResourceTypes
}

func (d *Device) GetData() any {
	return d.data
}

func (d *Device) GetProtocolIndependentID() string {
	return resources.ToUUID(d.cfg.ProtocolIndependentID).String()
}

type Config struct {
	ID                    string   `yaml:"id"`
	Name                  string   `yaml:"name"`
	ProtocolIndependentID string   `yaml:"protocolIndependentID"`
	ResourceTypes         []string `yaml:"resourceTypes"`
}

func (cfg *Config) Validate() error {
	if cfg.ProtocolIndependentID == "" {
		return fmt.Errorf("protocolIndependentID is required")
	}
	if cfg.ID == "" {
		cfg.ID = uuid.NewString()
	}

	if cfg.Name == "" {
		cfg.Name = "Unnamed"
	}
	return nil
}

func New(cfg Config, onDeviceUpdated func(d *Device), data any) *Device {
	cfg.ResourceTypes = resources.Unique(append(cfg.ResourceTypes, plgdDevice.ResourceType))
	d := &Device{
		cfg:             cfg,
		resources:       sync.NewMap[string, *resources.Resource](),
		onDeviceUpdated: onDeviceUpdated,
		data:            data,
	}
	return d
}

func (d *Device) AddResource(resource *resources.Resource) {
	d.resources.Store(resource.Href, resource)
}

type ConfigurationYaml struct {
	Manager *cloud.Manager `yaml:"cloudManager"`
}

func (d *Device) Encode(w io.Writer) error {
	dev := ConfigurationYaml{
		Manager: d.cloudManager,
	}
	return yaml.NewEncoder(w).Encode(dev)
}

func (d *Device) Decode(r io.Reader) error {
	dev := ConfigurationYaml{
		Manager: d.cloudManager,
	}
	err := yaml.NewDecoder(r).Decode(&dev)
	if err != nil {
		return err
	}
	d.cloudManager = dev.Manager
	return nil
}

func (d *Device) Init() {
	d.cloudManager.Init()
}

func (d *Device) SetCloudManager(uri string, requestHandler net.RequestHandler, maxMessageSize uint32) {
	d.cloudManager = cloud.New(d.cfg.ID, func() {
		d.onDeviceUpdated(d)
	}, requestHandler, d.GetLinksFilteredBy, maxMessageSize)
	d.AddResource(cloudResource.New(uri, d.cloudManager).Resource)
}

func (d *Device) UnregisterFromCloud() {
	if d.cloudManager != nil {
		d.cloudManager.Unregister()
	}
}

func (d *Device) Range(f func(key string, resource *resources.Resource) bool) {
	d.resources.Range(f)
}

func (d *Device) GetResource(key string) (*resources.Resource, bool) {
	return d.resources.Load(key)
}

func (d *Device) GetLinksFilteredBy(endpoints schema.Endpoints, deviceIDfilter string, resourceTypesFitler []string, policyBitMaskFitler schema.BitMask) (links schema.ResourceLinks) {
	di := deviceIDfilter
	if di != "" && di != d.GetID() {
		return nil
	}
	links = make(schema.ResourceLinks, 0, d.resources.Length())
	d.resources.Range(func(key string, resource *resources.Resource) bool {
		if len(resourceTypesFitler) > 0 && !resource.HasResourceTypes(resourceTypesFitler) {
			return true
		}
		if policyBitMaskFitler != 0 && resource.PolicyBitMask&policyBitMaskFitler == 0 {
			return true
		}
		links = append(links, schema.ResourceLink{
			Href:          key,
			ResourceTypes: resource.ResourceTypes,
			Interfaces:    resource.ResourceInterfaces,
			Policy: &schema.Policy{
				BitMask: resource.PolicyBitMask & (^resources.PublishToCloud),
			},
			Anchor:    "ocf://" + d.GetID(),
			DeviceID:  d.GetID(),
			Endpoints: endpoints,
		})
		return true
	})
	return links
}

func (d *Device) GetLinks(request *net.Request) (links schema.ResourceLinks) {
	return d.GetLinksFilteredBy(request.Endpoints, request.DeviceID(), request.ResourceTypes(), 0)
}

func (d *Device) RemoveResource(resource *resources.Resource) {
	d.resources.Delete(resource.Href)
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
