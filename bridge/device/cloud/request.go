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

package cloud

import (
	"bytes"
	"context"

	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/tcp/client"
)

func newRequestWithToken(ctx context.Context, c *client.Conn, uri string) (*pool.Message, error) {
	req := c.AcquireMessage(ctx)
	token, err := message.GetToken()
	if err != nil {
		return nil, err
	}
	req.SetToken(token)
	if err = req.SetPath(uri); err != nil {
		return nil, err
	}
	return req, nil
}

func newPostRequest(ctx context.Context, c *client.Conn, uri string, data interface{}) (*pool.Message, error) {
	req, err := newRequestWithToken(ctx, c, uri)
	if err != nil {
		return nil, err
	}
	req.SetCode(codes.POST)
	inputCbor, err := cbor.Encode(data)
	if err != nil {
		return nil, err
	}
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(inputCbor))
	return req, nil
}
