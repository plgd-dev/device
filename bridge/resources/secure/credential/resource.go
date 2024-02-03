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

package credential

import (
	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/schema/credential"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

type Resource struct {
	*resources.Resource
}

type Manager interface {
	Get(req *net.Request) (*pool.Message, error)
	Post(req *net.Request) (*pool.Message, error)
}

func New(uri string, m Manager) *Resource {
	d := &Resource{}
	d.Resource = resources.NewResource(uri, m.Get, m.Post, []string{credential.ResourceType}, []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_RW})
	// don't publish credential resource to cloud
	d.PolicyBitMask &= ^resources.PublishToCloud
	return d
}
