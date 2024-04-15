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

package thingDescription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/web-of-things-open-source/thingdescription-go/thingDescription"
)

const (
	ResourceURI  = "/.well-known/wot"
	ResourceType = "wot.thing"
)

var MessageTypeTDJson = message.MediaType(432) ///< application/td+json

type (
	GetThingDescription  func(ctx context.Context, endpoints schema.Endpoints) *thingDescription.ThingDescription
	RegisterSubscription func(subscription func(td *thingDescription.ThingDescription, closed bool)) func()
)

type Resource struct {
	*resources.Resource
	onGetThingDescription  GetThingDescription
	onRegisterSubscription RegisterSubscription
}

func (r *Resource) createMessage(request *net.Request, thingDescription *thingDescription.ThingDescription) (*pool.Message, error) {
	dataJson, err := json.Marshal(thingDescription)
	if err != nil {
		return resources.CreateErrorResponse(request.Context(), codes.InternalServerError, err)
	}
	mediaType, err := request.Accept()
	if err != nil {
		mediaType = MessageTypeTDJson
	}
	switch mediaType {
	case message.AppJSON, MessageTypeTDJson:
		res := pool.NewMessage(request.Context())
		res.SetCode(codes.Content)
		res.SetContentFormat(mediaType)
		res.SetBody(bytes.NewReader(dataJson))
		return res, nil
	case message.AppCBOR, message.AppOcfCbor:
		var v interface{}
		err := json.Unmarshal(dataJson, &v)
		if err != nil {
			return resources.CreateErrorResponse(request.Context(), codes.InternalServerError, err)
		}
		return resources.CreateResponseContent(request.Context(), v, codes.Content)
	}
	return resources.CreateErrorResponse(request.Context(), codes.NotAcceptable, errors.New("unsupported accept content format"))
}

func (r *Resource) Get(request *net.Request) (*pool.Message, error) {
	thingDescription := r.onGetThingDescription(request.Context(), request.Endpoints)
	if thingDescription == nil {
		return resources.CreateErrorResponse(request.Context(), codes.NotFound, errors.New("thing description not found"))
	}
	return r.createMessage(request, thingDescription)
}

func (r *Resource) CreateSubscription(req *net.Request, handler func(*pool.Message, error)) (func(), error) {
	unregister := r.onRegisterSubscription(func(td *thingDescription.ThingDescription, closed bool) {
		if closed {
			handler(nil, errors.New("subscription closed"))
			return
		}
		msg, err := r.createMessage(req, td)
		handler(msg, err)
	})
	return unregister, nil
}

func New(uri string, onGetThingDescription GetThingDescription, onRegisterSubscription RegisterSubscription) *Resource {
	r := &Resource{
		onGetThingDescription:  onGetThingDescription,
		onRegisterSubscription: onRegisterSubscription,
	}
	r.Resource = resources.NewResource(uri,
		r.Get,
		nil,
		[]string{ResourceType},
		[]string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_R},
	)
	return r
}
