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

package maintenance

import (
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

type OnFactoryReset func()

type Resource struct {
	*resources.Resource
	onFactoryReset OnFactoryReset
}

func (r *Resource) Get(request *net.Request) (*pool.Message, error) {
	factoryReset := false
	return resources.CreateResponseContent(request.Context(), maintenance.MaintenanceV1{
		FactoryReset: &factoryReset,
	}, codes.Content)
}

func (r *Resource) Post(request *net.Request) (*pool.Message, error) {
	var upd maintenance.MaintenanceUpdateRequest
	err := cbor.ReadFrom(request.Body(), &upd)
	if err != nil {
		return resources.CreateResponseBadRequest(request.Context(), err)
	}
	if upd.FactoryReset {
		r.onFactoryReset()
	}
	return resources.CreateResponseContent(request.Context(), maintenance.Maintenance{
		FactoryReset: false,
	}, codes.Changed)
}

func New(uri string, onFactoryReset OnFactoryReset) *Resource {
	r := &Resource{
		onFactoryReset: onFactoryReset,
	}
	r.Resource = resources.NewResource(uri,
		r.Get,
		r.Post,
		[]string{maintenance.ResourceType},
		[]string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_RW},
	)
	return r
}
