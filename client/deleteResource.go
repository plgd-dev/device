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

package client

import (
	"context"

	"github.com/plgd-dev/device/v2/client/core"
	codecOcf "github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
)

// DeleteResource deletes the resource from the device.
// In the absence of a cached device, it is found through multicast and stored with an expiration time.
func (c *Client) DeleteResource(
	ctx context.Context,
	deviceID string,
	href string,
	response interface{},
	opts ...DeleteOption,
) error {
	cfg := deleteOptions{
		codec:                  codecOcf.VNDOCFCBORCodec{},
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnDelete(cfg)
	}

	d, links, err := c.GetDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return err
	}

	link, err := core.GetResourceLink(links, href)
	if err != nil {
		return err
	}

	if c.useDeviceIDInQuery {
		cfg.opts = append(cfg.opts, coap.WithDeviceID(deviceID))
	}

	return d.DeleteResourceWithCodec(ctx, link, cfg.codec, response, cfg.opts...)
}
