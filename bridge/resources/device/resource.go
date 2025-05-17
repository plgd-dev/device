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

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

type Device interface {
	GetID() uuid.UUID
	GetName() string
	GetResourceTypes() []string
	GetProtocolIndependentID() uuid.UUID
}

type GetAdditionalPropertiesForResponseFunc func() map[string]interface{}

type Resource struct {
	*resources.Resource
	device                  Device
	getAdditionalProperties GetAdditionalPropertiesForResponseFunc
}

func New(uri string, dev Device, getAdditionalProperties GetAdditionalPropertiesForResponseFunc) *Resource {
	if getAdditionalProperties == nil {
		getAdditionalProperties = func() map[string]interface{} { return nil }
	}
	d := &Resource{
		device:                  dev,
		getAdditionalProperties: getAdditionalProperties,
	}
	d.Resource = resources.NewResource(uri, d.Get, nil, dev.GetResourceTypes(), []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_R})
	return d
}

func (d *Resource) Get(request *net.Request) (*pool.Message, error) {
	additionalProperties := d.getAdditionalProperties()
	deviceProperties := device.Device{
		ID:                    d.device.GetID().String(),
		Name:                  d.device.GetName(),
		ProtocolIndependentID: d.device.GetProtocolIndependentID().String(),
		// DataModelVersion:      "ocf.res.1.3.0",
		// SpecificationVersion:  "ocf.2.0.5",
	}
	if request.Interface() == interfaces.OC_IF_BASELINE {
		deviceProperties.ResourceTypes = d.GetResourceTypes()
		deviceProperties.Interfaces = d.ResourceInterfaces
	}
	properties := resources.MergeCBORStructs(additionalProperties, deviceProperties)
	res := pool.NewMessage(request.Context())
	res.SetCode(codes.Content)
	res.SetContentFormat(message.AppOcfCbor)
	data, err := cbor.Encode(properties)
	if err != nil {
		return nil, err
	}
	res.SetBody(bytes.NewReader(data))
	return res, nil
}
