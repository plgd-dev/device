// ************************************************************************
// Copyright (C) 2024 plgd.dev, s.r.o.
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
	"testing"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/pkg/codec/ocf"
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/stretchr/testify/require"
)

const (
	testIface = interfaces.OC_IF_BASELINE
	testQuery = "uid=nick.smith"
	testRt    = "oic.test"
)

func ifquery() string {
	return "if=" + interfaces.OC_IF_BASELINE
}

func rtquery() string {
	return "rt=" + testRt
}

func TestApplyOnCommonCommand(t *testing.T) {
	discoveryCfg := core.DiscoveryConfiguration{
		MulticastHopLimit: 42,
	}
	deviceID := "123"
	opts := []CommonCommandOption{
		WithDiscoveryConfiguration(discoveryCfg),
		WithQuery(testQuery),
		WithDeviceID(deviceID),
	}

	var o commonCommandOptions
	for _, opt := range opts {
		o = opt.applyOnCommonCommand(o)
	}

	// WithDiscoveryConfiguration
	require.Equal(t, discoveryCfg, o.discoveryConfiguration)

	mopts := message.Options{}
	for _, mopt := range o.opts {
		mopts = mopt(mopts)
	}

	queries, err := mopts.Queries()
	require.NoError(t, err)
	// WithQuery
	require.Contains(t, queries, testQuery)
	// WithDeviceID
	require.Contains(t, queries, "di="+deviceID)
}

func TestApplyOnGet(t *testing.T) {
	discoveryCfg := core.DiscoveryConfiguration{
		MulticastHopLimit: 42,
	}
	etag := "123"
	codec := ocf.VNDOCFCBORCodec{}
	linkNotFoundCallback := func(links schema.ResourceLinks, href string) (schema.ResourceLink, error) {
		return schema.ResourceLink{Href: href}, nil
	}
	opts := []GetOption{
		WithDiscoveryConfiguration(discoveryCfg),
		WithETag([]byte(etag)),
		WithInterface(testIface),
		WithQuery(testQuery),
		WithResourceTypes(testRt),
		WithCodec(codec),
		WithLinkNotFoundCallback(linkNotFoundCallback),
	}

	var o getOptions
	for _, opt := range opts {
		o = opt.applyOnGet(o)
	}

	// WithDiscoveryConfiguration
	require.Equal(t, discoveryCfg, o.discoveryConfiguration)
	// WithCodec
	require.Equal(t, codec, o.codec)
	// WithLinkNotFoundCallback
	require.NotNil(t, o.linkNotFoundCallback)
	notFoundTestLink := "/get/notfound"
	foundLink, err := o.linkNotFoundCallback(nil, notFoundTestLink)
	require.NoError(t, err)
	require.Equal(t, notFoundTestLink, foundLink.Href)

	mopts := message.Options{}
	for _, mopt := range o.opts {
		mopts = mopt(mopts)
	}

	// WithETag
	require.True(t, mopts.HasOption(message.ETag))
	val, err := mopts.GetBytes(message.ETag)
	require.NoError(t, err)
	require.Equal(t, etag, string(val))
	queries, err := mopts.Queries()
	require.NoError(t, err)
	// WithInterface
	require.Contains(t, queries, ifquery())
	// WithQuery
	require.Contains(t, queries, testQuery)
	// WithResourceTypes
	require.Contains(t, queries, rtquery())
}

func TestApplyOnObserve(t *testing.T) {
	discoveryCfg := core.DiscoveryConfiguration{
		MulticastHopLimit: 42,
	}
	codec := ocf.VNDOCFCBORCodec{}
	linkNotFoundCallback := func(links schema.ResourceLinks, href string) (schema.ResourceLink, error) {
		return schema.ResourceLink{Href: href}, nil
	}

	opts := []ObserveOption{
		WithDiscoveryConfiguration(discoveryCfg),
		WithInterface(testIface),
		WithQuery(testQuery),
		WithCodec(codec),
		WithLinkNotFoundCallback(linkNotFoundCallback),
	}

	var o observeOptions
	for _, opt := range opts {
		o = opt.applyOnObserve(o)
	}

	// WithDiscoveryConfiguration
	require.Equal(t, discoveryCfg, o.discoveryConfiguration)
	// WithCodec
	require.Equal(t, codec, o.codec)
	// WithLinkNotFoundCallback
	require.NotNil(t, o.linkNotFoundCallback)
	notFoundTestLink := "/observe/notfound"
	foundLink, err := o.linkNotFoundCallback(nil, notFoundTestLink)
	require.NoError(t, err)
	require.Equal(t, notFoundTestLink, foundLink.Href)

	mopts := message.Options{}
	for _, mopt := range o.opts {
		mopts = mopt(mopts)
	}

	queries, err := mopts.Queries()
	require.NoError(t, err)
	// WithInterface
	require.Contains(t, queries, ifquery())
	// WithQuery
	require.Contains(t, queries, testQuery)
}

func TestApplyOnUpdate(t *testing.T) {
	discoveryCfg := core.DiscoveryConfiguration{
		MulticastHopLimit: 42,
	}
	codec := ocf.VNDOCFCBORCodec{}
	linkNotFoundCallback := func(links schema.ResourceLinks, href string) (schema.ResourceLink, error) {
		return schema.ResourceLink{Href: href}, nil
	}
	opts := []UpdateOption{
		WithDiscoveryConfiguration(discoveryCfg),
		WithInterface(testIface),
		WithQuery(testQuery),
		WithCodec(codec),
		WithLinkNotFoundCallback(linkNotFoundCallback),
	}

	var o updateOptions
	for _, opt := range opts {
		o = opt.applyOnUpdate(o)
	}

	// WithDiscoveryConfiguration
	require.Equal(t, discoveryCfg, o.discoveryConfiguration)
	// WithCodec
	require.Equal(t, codec, o.codec)
	// WithLinkNotFoundCallback
	require.NotNil(t, o.linkNotFoundCallback)
	notFoundTestLink := "/update/notfound"
	foundLink, err := o.linkNotFoundCallback(nil, notFoundTestLink)
	require.NoError(t, err)
	require.Equal(t, notFoundTestLink, foundLink.Href)

	mopts := message.Options{}
	for _, mopt := range o.opts {
		mopts = mopt(mopts)
	}

	queries, err := mopts.Queries()
	require.NoError(t, err)
	// WithInterface
	require.Contains(t, queries, ifquery())
	// WithQuery
	require.Contains(t, queries, testQuery)
}

func TestApplyOnCreate(t *testing.T) {
	discoveryCfg := core.DiscoveryConfiguration{
		MulticastHopLimit: 42,
	}
	codec := ocf.VNDOCFCBORCodec{}
	opts := []CreateOption{
		WithDiscoveryConfiguration(discoveryCfg),
		WithQuery(testQuery),
		WithCodec(codec),
	}

	var o createOptions
	for _, opt := range opts {
		o = opt.applyOnCreate(o)
	}

	// WithDiscoveryConfiguration
	require.Equal(t, discoveryCfg, o.discoveryConfiguration)
	// WithCodec
	require.Equal(t, codec, o.codec)

	mopts := message.Options{}
	for _, mopt := range o.opts {
		mopts = mopt(mopts)
	}

	queries, err := mopts.Queries()
	require.NoError(t, err)
	// WithQuery
	require.Contains(t, queries, testQuery)
}

func TestApplyOnDelete(t *testing.T) {
	discoveryCfg := core.DiscoveryConfiguration{
		MulticastHopLimit: 42,
	}
	codec := ocf.VNDOCFCBORCodec{}
	linkNotFoundCallback := func(links schema.ResourceLinks, href string) (schema.ResourceLink, error) {
		return schema.ResourceLink{Href: href}, nil
	}
	opts := []DeleteOption{
		WithDiscoveryConfiguration(discoveryCfg),
		WithInterface(testIface),
		WithQuery(testQuery),
		WithCodec(codec),
		WithLinkNotFoundCallback(linkNotFoundCallback),
	}

	var o deleteOptions
	for _, opt := range opts {
		o = opt.applyOnDelete(o)
	}

	// WithDiscoveryConfiguration
	require.Equal(t, discoveryCfg, o.discoveryConfiguration)
	// WithCodec
	require.Equal(t, codec, o.codec)
	// WithLinkNotFoundCallback
	require.NotNil(t, o.linkNotFoundCallback)
	notFoundTestLink := "/delete/notfound"
	foundLink, err := o.linkNotFoundCallback(nil, notFoundTestLink)
	require.NoError(t, err)
	require.Equal(t, notFoundTestLink, foundLink.Href)

	mopts := message.Options{}
	for _, mopt := range o.opts {
		mopts = mopt(mopts)
	}

	queries, err := mopts.Queries()
	require.NoError(t, err)
	// WithInterface
	require.Contains(t, queries, ifquery())
	// WithQuery
	require.Contains(t, queries, testQuery)
}

func TestApplyOnOwn(t *testing.T) {
	discoveryCfg := core.DiscoveryConfiguration{
		MulticastHopLimit: 42,
	}
	psk := []byte("123")
	onOwn := func(context.Context, *coap.ClientCloseHandler) (string, error) {
		return "", nil
	}
	afterOwn := func(context.Context, *coap.ClientCloseHandler) error {
		return nil
	}
	opts := []OwnOption{
		WithDiscoveryConfiguration(discoveryCfg),
		WithPresharedKey(psk),
		WithActionDuringOwn(onOwn),
		WithActionAfterOwn(afterOwn),
		WithOTM(OTMType_Manufacturer),
		WithOTMs([]OTMType{OTMType_JustWorks}),
	}

	var o ownOptions
	for _, opt := range opts {
		o = opt.applyOnOwn(o)
	}

	// WithDiscoveryConfiguration
	require.Equal(t, discoveryCfg, o.discoveryConfiguration)
}

func TestApplyOnGetDevice(t *testing.T) {
	discoveryCfg := core.DiscoveryConfiguration{
		MulticastHopLimit: 42,
	}
	getDetails := func(context.Context, *core.Device, schema.ResourceLinks, ...func(message.Options) message.Options) (interface{}, error) {
		return "", nil
	}
	opts := []GetDeviceOption{
		WithDiscoveryConfiguration(discoveryCfg),
		WithQuery(testQuery),
		WithGetDetails(getDetails),
	}

	var o getDeviceOptions
	for _, opt := range opts {
		o = opt.applyOnGetDevice(o)
	}

	// WithDiscoveryConfiguration
	require.Equal(t, discoveryCfg, o.discoveryConfiguration)
	// WithGetDetails
	require.NotNil(t, o.getDetails)

	mopts := message.Options{}
	for _, mopt := range o.opts {
		mopts = mopt(mopts)
	}

	queries, err := mopts.Queries()
	require.NoError(t, err)
	// WithQuery
	require.Contains(t, queries, testQuery)
}

func TestApplyOnGetDevices(t *testing.T) {
	discoveryCfg := core.DiscoveryConfiguration{
		MulticastHopLimit: 42,
	}
	getDetails := func(context.Context, *core.Device, schema.ResourceLinks, ...func(message.Options) message.Options) (interface{}, error) {
		return "", nil
	}
	opts := []GetDevicesOption{
		WithDiscoveryConfiguration(discoveryCfg),
		WithGetDetails(getDetails),
		WithResourceTypes(testRt),
		WithUseDeviceID(true),
	}

	var o getDevicesOptions
	for _, opt := range opts {
		o = opt.applyOnGetDevices(o)
	}

	// WithDiscoveryConfiguration
	require.Equal(t, discoveryCfg, o.discoveryConfiguration)
	// WithGetDetails
	require.NotNil(t, o.getDetails)
	// WithResourceTypes
	require.Contains(t, o.resourceTypes, testRt)
	// WithUseDeviceID
	require.True(t, o.useDeviceID)
}

func TestApplyOnGetGetDevicesWithHandler(t *testing.T) {
	discoveryCfg := core.DiscoveryConfiguration{
		MulticastHopLimit: 42,
	}
	opts := []GetDevicesWithHandlerOption{
		WithDiscoveryConfiguration(discoveryCfg),
	}

	var o getDevicesWithHandlerOptions
	for _, opt := range opts {
		o = opt.applyOnGetGetDevicesWithHandler(o)
	}

	// WithDiscoveryConfiguration
	require.Equal(t, discoveryCfg, o.discoveryConfiguration)
}

func TestApplyOnGetDeviceByIP(t *testing.T) {
	getDetails := func(context.Context, *core.Device, schema.ResourceLinks, ...func(message.Options) message.Options) (interface{}, error) {
		return "", nil
	}
	opts := []GetDeviceByIPOption{
		WithGetDetails(getDetails),
		WithQuery(testQuery),
	}

	var o getDeviceByIPOptions
	for _, opt := range opts {
		o = opt.applyOnGetDeviceByIP(o)
	}

	// WithGetDetails
	require.NotNil(t, o.getDetails)

	mopts := message.Options{}
	for _, mopt := range o.opts {
		mopts = mopt(mopts)
	}

	queries, err := mopts.Queries()
	require.NoError(t, err)
	// WithQuery
	require.Contains(t, queries, testQuery)
}
