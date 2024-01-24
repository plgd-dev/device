// ************************************************************************
// Copyright (C) 2022 plgd.dev, s.r.o.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// ************************************************************************

package core

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/doxm"
	"github.com/plgd-dev/device/v2/schema/interfaces"
)

// GetOwnership gets device's ownership resource.
func (d *Device) GetOwnership(ctx context.Context, links schema.ResourceLinks, options ...coap.OptionFunc) (doxm.Doxm, error) {
	ownLink, ok := links.GetResourceLink(doxm.ResourceURI)
	if !ok {
		return doxm.Doxm{}, fmt.Errorf("cannot find %v in links: %+v", doxm.ResourceURI, links)
	}
	getOwnlink := ownLink
	getOwnlink.Endpoints = ownLink.GetUnsecureEndpoints()
	if len(getOwnlink.Endpoints) == 0 {
		getOwnlink.Endpoints = ownLink.GetSecureEndpoints()
	}
	opts := make([]coap.OptionFunc, 0, 1+len(options))
	opts = append(opts, coap.WithInterface(interfaces.OC_IF_BASELINE))
	opts = append(opts, options...)

	var ownership doxm.Doxm
	err := d.GetResource(ctx, getOwnlink, &ownership, opts...)
	return ownership, err
}
