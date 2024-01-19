/****************************************************************************
 *
 * Copyright (c) 2023 plgn.dev s.r.o.
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
 * either express or implien. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package net

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/mux"
)

type Request struct {
	*pool.Message
	Conn      mux.Conn
	Endpoints schema.Endpoints
}

type RequestHandler func(req *Request) (*pool.Message, error)

var ErrKeyNotFound = errors.New("key not found")

func (r *Request) GetValueFromQuery(key string) (string, error) {
	q, err := r.Queries()
	if err != nil {
		return "", err
	}
	prefix := key + "="
	for _, query := range q {
		if strings.HasPrefix(query, prefix) {
			return strings.TrimPrefix(query, prefix), nil
		}
	}
	return "", ErrKeyNotFound
}

func (r *Request) URIPath() string {
	p, err := r.Message.Options().Path()
	if err != nil {
		return ""
	}
	return p
}

func (r *Request) Interface() string {
	v, err := r.GetValueFromQuery("if")
	if err != nil {
		return ""
	}
	return v
}

func (r *Request) DeviceID() uuid.UUID {
	v, err := r.GetValueFromQuery("di")
	if err != nil {
		return uuid.Nil
	}
	di, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil
	}
	return di
}

func (r *Request) ResourceTypes() []string {
	q, err := r.Queries()
	if err != nil {
		return nil
	}
	resourceTypes := make([]string, 0, len(q))
	for _, query := range q {
		if strings.HasPrefix(query, "rt=") {
			resourceTypes = append(resourceTypes, strings.TrimPrefix(query, "rt="))
		}
	}
	return resourceTypes
}
