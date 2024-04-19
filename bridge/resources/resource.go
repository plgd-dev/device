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

package resources

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc64"
	"io"
	"reflect"

	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/device/v2/pkg/eventloop"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/pkg/sync"
	"go.uber.org/atomic"
)

type GetHandlerFunc func(*net.Request) (*pool.Message, error)

type PostHandlerFunc func(*net.Request) (*pool.Message, error)

type CreateSubscriptionFunc func(*net.Request, func(*pool.Message, error)) (func(), error)

const PublishToCloud schema.BitMask = 1 << 7

type subscription struct {
	done   <-chan struct{}
	cancel func()
}

type Resource struct {
	Href                string
	ResourceInterfaces  []string
	PolicyBitMask       schema.BitMask
	getHandler          GetHandlerFunc
	postHandler         PostHandlerFunc
	createSubscription  CreateSubscriptionFunc
	closed              atomic.Bool
	createdSubscription *sync.Map[string, *subscription]
	etag                atomic.Uint64
	loop                *eventloop.Loop
	resourceTypes       atomic.Pointer[[]string]
}

func (r *Resource) GetPolicyBitMask() schema.BitMask {
	return r.PolicyBitMask
}

func (r *Resource) GetHref() string {
	return r.Href
}

type SupportedOperation int

const (
	SupportedOperationRead SupportedOperation = 0x1 << iota
	SupportedOperationWrite
	SupportedOperationObserve
)

func (o SupportedOperation) HasOperation(operation SupportedOperation) bool {
	return o&operation != 0
}

func (r *Resource) SupportsOperations() SupportedOperation {
	var operations SupportedOperation
	if r.getHandler != nil {
		operations |= SupportedOperationRead
	}
	if r.postHandler != nil {
		operations |= SupportedOperationWrite
	}
	if r.PolicyBitMask&schema.Observable != 0 {
		operations |= SupportedOperationObserve
	}
	return operations
}

func (r *Resource) SetResourceTypes(resourceTypes []string) {
	resourceTypes = Unique(resourceTypes)
	r.resourceTypes.Store(&resourceTypes)
}

func (r *Resource) GetResourceTypes() []string {
	resourceTypes := r.resourceTypes.Load()
	if resourceTypes == nil {
		return nil
	}
	return *resourceTypes
}

func (r *Resource) GetResourceInterfaces() []string {
	return r.ResourceInterfaces
}

func Unique(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func NewResource(href string, getHandler GetHandlerFunc, postHandler PostHandlerFunc, resourceTypes, resourceInterfaces []string) *Resource {
	r := &Resource{
		Href:                href,
		ResourceInterfaces:  Unique(resourceInterfaces),
		PolicyBitMask:       schema.Discoverable | PublishToCloud,
		getHandler:          getHandler,
		postHandler:         postHandler,
		createdSubscription: sync.NewMap[string, *subscription](),
	}
	r.SetResourceTypes(resourceTypes)
	r.etag.Store(GetETag())
	return r
}

func createTextPlainResponse(ctx context.Context, token message.Token, code codes.Code, body []byte) *pool.Message {
	msg := pool.NewMessage(ctx)
	msg.SetCode(code)
	if token != nil {
		msg.SetToken(token)
	}
	msg.SetContentFormat(message.TextPlain)
	msg.SetBody(bytes.NewReader(body))
	return msg
}

func CreateResponseMethodNotAllowed(ctx context.Context, token message.Token) *pool.Message {
	return createTextPlainResponse(ctx, token, codes.MethodNotAllowed, []byte(fmt.Sprintf("unsupported method %v", codes.MethodNotAllowed)))
}

func CreateResponseContentWithCodec(ctx context.Context, codec coap.Codec, data interface{}, code codes.Code) (*pool.Message, error) {
	d, err := codec.Encode(data)
	if err != nil {
		return nil, err
	}
	res := pool.NewMessage(ctx)
	res.SetCode(code)
	res.SetContentFormat(codec.ContentFormat())
	res.SetBody(bytes.NewReader(d))
	return res, nil
}

func CreateResponseContent(ctx context.Context, data interface{}, code codes.Code) (*pool.Message, error) {
	if str, ok := data.(string); ok {
		return createTextPlainResponse(ctx, nil, code, []byte(str)), nil
	}
	return CreateResponseContentWithCodec(ctx, ocf.VNDOCFCBORCodec{}, data, code)
}

func CreateErrorResponse(ctx context.Context, code codes.Code, err error) (*pool.Message, error) {
	return createTextPlainResponse(ctx, nil, code, []byte(err.Error())), nil
}

func CreateResponseBadRequest(ctx context.Context, err error) (*pool.Message, error) {
	return CreateErrorResponse(ctx, codes.BadRequest, err)
}

func (r *Resource) SetObserveHandler(loop *eventloop.Loop, createSubscription CreateSubscriptionFunc) {
	if createSubscription == nil {
		r.createSubscription = nil
		r.PolicyBitMask &^= schema.Observable
		r.loop = nil
		return
	}
	r.loop = loop
	r.createSubscription = createSubscription
	r.PolicyBitMask |= schema.Observable
}

// Close closes resource and cancel all subscriptions
func (r *Resource) Close() {
	if !r.closed.CompareAndSwap(false, true) {
		return
	}
	r.createdSubscription.Range(func(_ string, value *subscription) bool {
		value.cancel()
		return true
	})
}

func (r *Resource) removeSubscription(key string) {
	sub, ok := r.createdSubscription.LoadAndDelete(key)
	if ok {
		sub.cancel()
	}
}

func (r *Resource) ETag() []byte {
	etag := r.etag.Load()
	if etag == 0 {
		return nil
	}
	e := make([]byte, 8)
	binary.BigEndian.PutUint64(e, etag)
	return e
}

func (r *Resource) UpdateETag() {
	r.etag.Store(GetETag())
}

func calcCRC64(body io.ReadSeeker) uint64 {
	if body == nil {
		return 0
	}
	h := crc64.New(crc64.MakeTable(crc64.ISO))
	_, _ = io.Copy(h, body)
	_, _ = body.Seek(0, io.SeekStart)
	return h.Sum64()
}

func (r *Resource) observerHandler(req *net.Request, createSubscription bool) (*pool.Message, error) {
	if r.loop == nil {
		return CreateErrorResponse(req.Context(), codes.InternalServerError, errors.New("event loop is not initialized"))
	}
	if !createSubscription {
		r.removeSubscription(req.Conn.RemoteAddr().String())
		return r.getHandler(req)
	}
	req.Message.Hijack()
	sequence := atomic.NewUint32(1)
	var cancel func()
	var err error
	var deduplicationNotification atomic.Uint64
	cancel, err = r.createSubscription(req, func(resp *pool.Message, err error) {
		if err == nil {
			d := calcCRC64(resp.Body())
			if deduplicationNotification.Swap(d) == d {
				return
			}
			resp.SetObserve(sequence.Inc())
		} else {
			defer r.removeSubscription(req.Conn.RemoteAddr().String())
			resp, err = CreateResponseBadRequest(req.Conn.Context(), fmt.Errorf("error while observing %s: %w", r.Href, err))
			if err != nil {
				return
			}
		}
		resp.SetContext(req.Conn.Context())
		resp.SetToken(req.Token())
		etag := r.ETag()
		if etag != nil {
			_ = resp.SetETag(etag)
		}

		err = req.Conn.WriteMessage(resp)
		if err != nil {
			r.removeSubscription(req.Conn.RemoteAddr().String())
			return
		}
	})
	if err != nil {
		return CreateResponseBadRequest(req.Context(), err)
	}
	resp, err := r.getHandler(req)
	if err != nil {
		cancel()
		return nil, err
	}
	// set deduplicationNotification value to current value of body
	d := calcCRC64(resp.Body())
	deduplicationNotification.Store(d)
	resp.SetToken(req.Token())
	resp.SetObserve(sequence.Inc())
	oldSub, oldLoaded := r.createdSubscription.Replace(req.Conn.RemoteAddr().String(), &subscription{
		done:   req.Context().Done(),
		cancel: cancel,
	})
	if oldLoaded {
		oldSub.cancel()
	}
	r.loop.Add(eventloop.NewReadHandler(reflect.ValueOf(req.Context().Done()), func(_ reflect.Value, closed bool) {
		if closed {
			r.removeSubscription(req.Conn.RemoteAddr().String())
			return
		}
	}))
	return resp, nil
}

func (r *Resource) HandleRequest(req *net.Request) (*pool.Message, error) {
	if req.Code() == codes.GET && r.getHandler != nil { //nolint:nestif
		var resp *pool.Message
		var err error
		if obs, errObs := req.Observe(); errObs == nil && r.createSubscription != nil && r.PolicyBitMask&schema.Observable != 0 {
			resp, err = r.observerHandler(req, obs == 0)
		} else {
			resp, err = r.getHandler(req)
		}
		if resp != nil && resp.Code() == codes.Content {
			etag := r.ETag()
			if etag != nil {
				_ = resp.SetETag(etag)
			}
		}
		return resp, err
	}
	if req.Code() == codes.POST && r.postHandler != nil {
		return r.postHandler(req)
	}
	return CreateResponseMethodNotAllowed(req.Context(), req.Token()), nil
}
