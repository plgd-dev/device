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
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/plgd-dev/device/v2/client/core"
	codecOcf "github.com/plgd-dev/device/v2/pkg/codec/ocf"
	pkgError "github.com/plgd-dev/device/v2/pkg/error"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
)

type observerCodec struct {
	contentFormat message.MediaType
}

// ContentFormat propagates the CoAP media type.
func (c observerCodec) ContentFormat() message.MediaType { return c.contentFormat }

// Encode propagates the payload without any conversions.
func (c observerCodec) Encode(interface{}) ([]byte, error) {
	return nil, pkgError.NotSupported()
}

// Decode validates the content format and
// propagates the payload to v as *[]byte without any conversions.
func (c observerCodec) Decode(m *pool.Message, v interface{}) error {
	if v == nil {
		return nil
	}
	if m.Code() != codes.Valid && m.Body() == nil {
		return fmt.Errorf("unexpected empty body")
	}
	p, ok := v.(**pool.Message)
	if !ok {
		return fmt.Errorf("expected **pool.Message instead of %T", v)
	}
	*p = m
	return nil
}

type observationsHandler struct {
	client *Client
	device *core.Device
	id     string

	sync.Mutex

	observationID string
	lastMessage   atomic.Value

	observations *coapSync.Map[string, *observationHandler]
}

type decodeFunc = func(v interface{}, codec coap.Codec) error

type observationHandler struct {
	handler      core.ObservationHandler
	codec        coap.Codec
	lock         sync.Mutex
	isClosed     bool
	firstMessage decodeFunc
}

func createDecodeFunc(message *pool.Message) decodeFunc {
	var l sync.Mutex
	return func(v interface{}, codec coap.Codec) error {
		l.Lock()
		defer l.Unlock()
		v, err := coap.TrySetDetailedReponse(message, v)
		if err != nil {
			return err
		}
		switch code := message.Code(); {
		case code == codes.Content:
			_, err := message.Body().Seek(0, io.SeekStart)
			if err != nil {
				return err
			}
			return codec.Decode(message, v)
		case code == codes.Valid:
			return nil
		}
		return fmt.Errorf("request failed: %s", codecOcf.Dump(message))
	}
}

func (h *observationHandler) handleMessageLocked(ctx context.Context, decode decodeFunc) {
	if decode == nil {
		return
	}
	if h.isClosed {
		return
	}

	h.handler.Handle(ctx, func(v interface{}) error {
		return decode(v, h.codec)
	})
}

func (h *observationHandler) HandleMessage(ctx context.Context, decode decodeFunc) {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.firstMessage = nil
	h.handleMessageLocked(ctx, decode)
}

func (h *observationHandler) HandleFirstMessage() {
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.firstMessage == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.handleMessageLocked(ctx, h.firstMessage)
}

func (h *observationHandler) OnClose() {
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.isClosed {
		return
	}
	h.isClosed = true
	h.handler.OnClose()
}

func (h *observationHandler) Error(err error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.isClosed {
		return
	}
	h.isClosed = true
	h.handler.Error(err)
}

func getObservationID(resourceCacheID, resourceObservationID string) string {
	return strings.Join([]string{resourceCacheID, resourceObservationID}, "/")
}

func parseIDs(id string) (string, string, error) {
	v := strings.Split(id, "/")
	if len(v) != 2 {
		return "", "", fmt.Errorf("invalid ID")
	}
	return v[0], v[1], nil
}

// ObserveResource method starts observing the resource of the device.
// In the absence of a cached device, it is found through multicast and stored with an expiration time.
func (c *Client) ObserveResource(
	ctx context.Context,
	deviceID string,
	href string,
	handler core.ObservationHandler,
	opts ...ObserveOption,
) (observationID string, _ error) {
	cfg := observeOptions{
		codec:                  codecOcf.VNDOCFCBORCodec{},
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnObserve(cfg)
	}
	resourceObservationID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	key := uuid.NewSHA1(uuid.NameSpaceURL, []byte(deviceID+href+"?if="+cfg.resourceInterface)).String()
	h, loaded := c.observeResourceCache.LoadOrStoreWithFunc(key, func(h *observationsHandler) *observationsHandler {
		h.Lock()
		return h
	}, func() *observationsHandler {
		h := observationsHandler{
			observations: coapSync.NewMap[string, *observationHandler](),
			client:       c,
			id:           key,
		}
		h.Lock()
		return &h
	})
	defer h.Unlock()
	lastMessage := h.lastMessage.Load()
	var firstMessage decodeFunc
	if lastMessage != nil {
		firstMessage = lastMessage.(decodeFunc) //nolint:forcetypeassert
	}

	obsHandler := observationHandler{
		handler:      handler,
		codec:        cfg.codec,
		firstMessage: firstMessage,
	}
	h.observations.Store(resourceObservationID.String(), &obsHandler)
	if loaded {
		go obsHandler.HandleFirstMessage()
		return getObservationID(key, resourceObservationID.String()), nil
	}

	d, links, err := c.GetDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return "", err
	}

	link, err := core.GetResourceLink(links, href)
	if err != nil {
		return "", err
	}

	observationID, err = d.ObserveResourceWithCodec(ctx, link, observerCodec{contentFormat: cfg.codec.ContentFormat()}, h, cfg.opts...)
	if err != nil {
		return "", err
	}

	dev, _ := c.deviceCache.UpdateOrStoreDevice(d)
	h.observationID = observationID
	h.device = dev

	return getObservationID(key, resourceObservationID.String()), err
}

// StopObservingResource method stops observing the resource of the device.
// In the absence of a cached device, it is found through multicast and stored with an expiration time.
func (c *Client) StopObservingResource(ctx context.Context, observationID string) (bool, error) {
	resourceCacheID, internalResourceObservationID, err := parseIDs(observationID)
	if err != nil {
		return false, err
	}
	var resourceObservationID string
	var dev *core.Device
	c.observeResourceCache.ReplaceWithFunc(resourceCacheID, func(oldValue *observationsHandler, oldLoaded bool) (newValue *observationsHandler, deleteHandler bool) {
		if !oldLoaded {
			return nil, true
		}
		h := oldValue
		resourceObservationID = h.observationID
		_, ok := h.observations.LoadAndDelete(internalResourceObservationID)
		if !ok {
			return h, false
		}

		if h.observations.Length() == 0 {
			dev = h.device
			return nil, true
		}
		return h, false
	})
	if dev == nil {
		return false, nil
	}
	_ = c.deviceCache.TryToChangeDeviceExpirationToDefault(dev.DeviceID())
	ok, err := dev.StopObservingResource(ctx, resourceObservationID)
	if err != nil {
		return false, fmt.Errorf("failed to stop resource observation(%s) in device(%s): %w", observationID, dev.DeviceID(), err)
	}

	return ok, nil
}

func (c *Client) closeObservingResource(ctx context.Context, o *observationsHandler) {
	_, ok := c.observeResourceCache.LoadAndDelete(o.id)
	if !ok {
		return
	}
	o.Lock()
	defer o.Unlock()
	if o.device != nil {
		deviceID := o.device.DeviceID()
		if _, err := o.device.StopObservingResource(ctx, o.observationID); err != nil {
			c.logger.Warn(fmt.Errorf("failed to stop resources observation in device(%s): %w", deviceID, err).Error())
		}
		_ = c.deviceCache.TryToChangeDeviceExpirationToDefault(deviceID)
	}
}

func (o *observationsHandler) Handle(ctx context.Context, body coap.DecodeFunc) {
	var message *pool.Message
	err := body(&message)
	if err != nil {
		o.Error(err)
		return
	}
	decode := createDecodeFunc(message)
	o.lastMessage.Store(decode)
	o.observations.Range(func(key string, h *observationHandler) bool {
		h.HandleMessage(ctx, decode)
		return true
	})
}

func (o *observationsHandler) OnClose() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o.client.closeObservingResource(ctx, o)
	for _, h := range o.observations.LoadAndDeleteAll() {
		h.handler.OnClose()
	}
}

func (o *observationsHandler) Error(err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o.client.closeObservingResource(ctx, o)
	for _, h := range o.observations.LoadAndDeleteAll() {
		h.handler.Error(err)
	}
}
