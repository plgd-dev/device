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

package service

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/plgd-dev/device/v2/pkg/ocf/cloud"
	coapCodes "github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/mux"
	"github.com/plgd-dev/kit/v2/codec/cbor"
)

// fixHref ensures that href starts with "/" and does not end with "/".
func fixHref(href string) string {
	backslash := regexp.MustCompile(`\/+`)
	p := backslash.ReplaceAllString(href, "/")
	p = strings.TrimRight(p, "/")
	if len(p) > 0 && p[0] == '/' {
		return p
	}
	return "/" + p
}

func publishHandler(req *mux.Message, client *Client) {
	p := cloud.PublishResourcesRequest{
		TimeToLive: -1,
	}
	err := cbor.ReadFrom(req.Body(), &p)
	if err != nil {
		client.sendErrorResponse(fmt.Errorf("cannot read publish request body received: %w", err), coapCodes.BadRequest, req.Token())
		return
	}

	for i, link := range p.Links {
		p.Links[i].DeviceID = p.DeviceID
		p.Links[i].Href = fixHref(link.Href)
	}

	if err = client.handler.PublishResources(p); err != nil {
		client.sendErrorResponse(err, coapCodes.InternalServerError, req.Token())
		return
	}

	out, err := cbor.Encode(p)
	if err != nil {
		client.sendErrorResponse(err, coapCodes.InternalServerError, req.Token())
		return
	}

	client.sendResponse(coapCodes.Changed, req.Token(), out)
}

func parseUnpublishRequestFromQuery(queries []string) (cloud.UnpublishResourcesRequest, error) {
	req := cloud.UnpublishResourcesRequest{}
	for _, q := range queries {
		values, err := url.ParseQuery(q)
		if err != nil {
			return cloud.UnpublishResourcesRequest{}, fmt.Errorf("cannot parse unpublish query: %w", err)
		}
		if di := values.Get("di"); di != "" {
			req.DeviceID = di
		}

		if ins := values.Get("ins"); ins != "" {
			i, err := strconv.Atoi(ins)
			if err != nil {
				return cloud.UnpublishResourcesRequest{}, fmt.Errorf("cannot convert %v to number", ins)
			}
			req.InstanceIDs = append(req.InstanceIDs, int64(i))
		}
	}

	if req.DeviceID == "" {
		return cloud.UnpublishResourcesRequest{}, fmt.Errorf("deviceID not found")
	}
	return req, nil
}

func unpublishHandler(req *mux.Message, client *Client) {
	queries, err := req.Options().Queries()
	if err != nil {
		client.sendErrorResponse(fmt.Errorf("cannot query string from unpublish request from device %v: %w", client.GetDeviceID(), err), coapCodes.BadRequest, req.Token())
		return
	}

	r, err := parseUnpublishRequestFromQuery(queries)
	if err != nil {
		client.sendErrorResponse(fmt.Errorf("unable to parse unpublish request query string from device %v: %w", client.GetDeviceID(), err), coapCodes.BadRequest, req.Token())
		return
	}

	err = client.handler.UnpublishResources(r)
	if err != nil {
		client.sendErrorResponse(err, coapCodes.InternalServerError, req.Token())
		return
	}

	client.sendResponse(coapCodes.Deleted, req.Token(), nil)
}

func resourceDirectoryHandler(req *mux.Message, client *Client) {
	switch req.Code() {
	case coapCodes.POST:
		publishHandler(req, client)
	case coapCodes.DELETE:
		unpublishHandler(req, client)
	default:
		client.sendErrorResponse(fmt.Errorf("forbidden request from %v", client.RemoteAddrString()), coapCodes.Forbidden, req.Token())
	}
}
