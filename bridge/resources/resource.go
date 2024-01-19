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
	"fmt"
	"hash/crc64"
	"io"
	"reflect"

	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/pkg/sync"
	"go.uber.org/atomic"
)

type GetHandlerFunc func(req *net.Request) (*pool.Message, error)

type PostHandlerFunc func(req *net.Request) (*pool.Message, error)

type CreateSubscriptionFunc func(req *net.Request, handler func(msg *pool.Message, err error)) (cancel func(), err error)

const PublishToCloud schema.BitMask = 1 << 7

type subscription struct {
	done   <-chan struct{}
	cancel func()
}

type Resource struct {
	Href                string
	ResourceTypes       []string
	ResourceInterfaces  []string
	PolicyBitMask       schema.BitMask
	getHandler          GetHandlerFunc
	postHandler         PostHandlerFunc
	createSubscription  CreateSubscriptionFunc
	closed              atomic.Bool
	wakeUpSubscription  chan bool
	createdSubscription *sync.Map[string, *subscription]
	etag                atomic.Uint64
}

func (r *Resource) GetPolicyBitMask() schema.BitMask {
	return r.PolicyBitMask
}

func (r *Resource) GetHref() string {
	return r.Href
}

func (r *Resource) GetResourceTypes() []string {
	return r.ResourceTypes
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
		ResourceTypes:       Unique(resourceTypes),
		ResourceInterfaces:  Unique(resourceInterfaces),
		PolicyBitMask:       schema.Discoverable | PublishToCloud,
		getHandler:          getHandler,
		postHandler:         postHandler,
		wakeUpSubscription:  make(chan bool, 1),
		createdSubscription: sync.NewMap[string, *subscription](),
	}
	r.etag.Store(GetETag())
	go r.watchSubscriptions()
	return r
}

func CreateResponseMethodNotAllowed(ctx context.Context, token message.Token) *pool.Message {
	msg := pool.NewMessage(ctx)
	msg.SetCode(codes.MethodNotAllowed)
	msg.SetToken(token)
	msg.SetContentFormat(message.TextPlain)
	msg.SetBody(bytes.NewReader([]byte(fmt.Sprintf("unsupported method %v", codes.MethodNotAllowed))))
	return msg
}

func CreateResponseContent(ctx context.Context, data interface{}, code codes.Code) (*pool.Message, error) {
	if str, ok := data.(string); ok {
		res := pool.NewMessage(ctx)
		res.SetCode(code)
		res.SetContentFormat(message.TextPlain)
		res.SetBody(bytes.NewReader([]byte(str)))
		return res, nil
	}
	d, err := cbor.Encode(data)
	if err != nil {
		return nil, err
	}
	res := pool.NewMessage(ctx)
	res.SetCode(code)
	res.SetContentFormat(message.AppOcfCbor)
	res.SetBody(bytes.NewReader(d))
	return res, nil
}

func CreateResponseBadRequest(ctx context.Context, err error) (*pool.Message, error) {
	res := pool.NewMessage(ctx)
	res.SetCode(codes.BadRequest)
	res.SetContentFormat(message.TextPlain)
	res.SetBody(bytes.NewReader([]byte(err.Error())))
	return res, nil
}

func (r *Resource) SetObserveHandler(createSubscription CreateSubscriptionFunc) {
	if createSubscription == nil {
		r.createSubscription = nil
		r.PolicyBitMask &^= schema.Observable
		return
	}
	r.createSubscription = createSubscription
	r.PolicyBitMask |= schema.Observable
}

func (r *Resource) wakeWatchSubscriptions() {
	select {
	case r.wakeUpSubscription <- true:
	default:
	}
}

// Close closes resource and cancel all subscriptions
func (r *Resource) Close() {
	if !r.closed.CompareAndSwap(false, true) {
		return
	}
	r.createdSubscription.Range(func(key string, value *subscription) bool {
		value.cancel()
		return true
	})
	r.wakeUpSubscription <- false
}

func (r *Resource) watchSubscriptions() {
	for {
		keys := make([]string, 0, r.createdSubscription.Length())
		cases := make([]reflect.SelectCase, 0, r.createdSubscription.Length()+1)
		// wake up subscription
		cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(r.wakeUpSubscription)})
		r.createdSubscription.Range(func(key string, value *subscription) bool {
			cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(value.done)})
			keys = append(keys, key)
			return true
		})
		idx, recv, ok := reflect.Select(cases)
		if idx == 0 {
			if ok && recv.Bool() {
				// wake up subscription - added/
				continue
			}
			// resource closed
			// cancel all subscriptions
			r.createdSubscription.Range(func(key string, value *subscription) bool {
				value.cancel()
				return true
			})
			return
		}
		r.removeSubscription(keys[idx-1])
	}
}

func (r *Resource) removeSubscription(key string) {
	sub, ok := r.createdSubscription.LoadAndDelete(key)
	if ok {
		sub.cancel()
		r.wakeWatchSubscriptions()
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
	resp.SetToken(req.Token())
	resp.SetObserve(sequence.Inc())
	oldSub, oldLoaded := r.createdSubscription.Replace(req.Conn.RemoteAddr().String(), &subscription{
		done:   req.Context().Done(),
		cancel: cancel,
	})
	if oldLoaded {
		oldSub.cancel()
	}
	r.wakeWatchSubscriptions()
	return resp, nil
}

func (r *Resource) HandleRequest(req *net.Request) (*pool.Message, error) {
	if req.Code() == codes.GET && r.getHandler != nil {
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
