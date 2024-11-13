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
	"github.com/plgd-dev/device/v2/pkg/net/coap"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message"
)

// WithQuery updates/gets resource with a query directly from a device.
func WithQuery(resourceQuery string) ResourceQueryOption {
	return ResourceQueryOption{
		resourceQuery: resourceQuery,
	}
}

// WithDeviceID updates/gets resource with deviceID also in a query parameter for the bridged devices.
// Note: it is not needed when client has been created with UseDeviceIDInQuery.
func WithDeviceID(deviceID string) ResourceQueryOption {
	return ResourceQueryOption{
		resourceQuery: "di=" + deviceID,
	}
}

// WithInterface updates/gets resource with interface directly from a device.
func WithInterface(resourceInterface string) ResourceInterfaceOption {
	return ResourceInterfaceOption{
		resourceInterface: resourceInterface,
	}
}

func WithGetDetails(getDetails func(ctx context.Context, d *core.Device, links schema.ResourceLinks, optsArgs ...func(message.Options) message.Options) (interface{}, error)) GetDetailsOption {
	return GetDetailsOption{
		getDetails: getDetails,
	}
}

// WithUseDeviceID appends deviceID to all get requests during discovery.
// Note: it is not needed when client has been created with UseDeviceIDInQuery.
func WithUseDeviceID(useDeviceID bool) UseDeviceIDInQueryOption {
	return UseDeviceIDInQueryOption{
		UseDeviceID: useDeviceID,
	}
}

func WithCodec(codec coap.Codec) CodecOption {
	return CodecOption{
		codec: codec,
	}
}

func WithResourceTypes(resourceTypes ...string) ResourceTypesOption {
	return ResourceTypesOption{
		resourceTypes: resourceTypes,
	}
}

func WithETag(etag []byte) ResourceETagOption {
	return ResourceETagOption{
		etag: etag,
	}
}

// WithActionDuringOwn allows to set deviceID of owned device and other staff over owner TLS.
// returns new deviceID, if it returns error device will be disowned.
func WithActionDuringOwn(actionDuringOwn func(ctx context.Context, client *coap.ClientCloseHandler) (string, error)) OwnOption {
	return actionDuringOwnOption{
		actionDuringOwn: actionDuringOwn,
	}
}

// WithActionAfterOwn allows initialize configuration at the device via DTLS connection with preshared key. For example setup time / NTP.
// if it returns error device will be disowned.
func WithActionAfterOwn(actionAfterOwn func(ctx context.Context, client *coap.ClientCloseHandler) error) OwnOption {
	return actionAfterOwnOption{
		actionAfterOwn: actionAfterOwn,
	}
}

// WithPresharedKey allows to set preshared key for owner. It is not set, it will be randomized.
func WithPresharedKey(presharedKey []byte) OwnOption {
	return presharedKeyOption{
		presharedKey: presharedKey,
	}
}

// WithOTMs allows to set ownership transfer methods, by default it is []OTMType{manufacturer}. For owning, the first match in order of OTMType with the device will be used.
func WithOTMs(otmTypes []OTMType) OwnOption {
	return otmOption{
		otmTypes: otmTypes,
	}
}

// WithOTM allows to set ownership transfer method, by default it is manufacturer.
func WithOTM(otmType OTMType) OwnOption {
	return otmOption{
		otmTypes: []OTMType{otmType},
	}
}

type UseDeviceIDInQueryOption struct {
	UseDeviceID bool
}

func (r UseDeviceIDInQueryOption) applyOnGetDevices(opts getDevicesOptions) getDevicesOptions {
	opts.useDeviceID = r.UseDeviceID
	return opts
}

type ResourceETagOption struct {
	etag []byte
}

func (r ResourceETagOption) applyOnGet(opts getOptions) getOptions {
	if r.etag != nil {
		opts.opts = append(opts.opts, coap.WithETag(r.etag))
	}
	return opts
}

type ResourceInterfaceOption struct {
	resourceInterface string
}

func (r ResourceInterfaceOption) applyOnGet(opts getOptions) getOptions {
	if r.resourceInterface != "" {
		opts.opts = append(opts.opts, coap.WithInterface(r.resourceInterface))
	}
	return opts
}

func (r ResourceInterfaceOption) applyOnObserve(opts observeOptions) observeOptions {
	if r.resourceInterface != "" {
		opts.opts = append(opts.opts, coap.WithInterface(r.resourceInterface))
		opts.resourceInterface = r.resourceInterface
	}
	return opts
}

func (r ResourceInterfaceOption) applyOnUpdate(opts updateOptions) updateOptions {
	if r.resourceInterface != "" {
		opts.opts = append(opts.opts, coap.WithInterface(r.resourceInterface))
	}
	return opts
}

func (r ResourceInterfaceOption) applyOnDelete(opts deleteOptions) deleteOptions {
	if r.resourceInterface != "" {
		opts.opts = append(opts.opts, coap.WithInterface(r.resourceInterface))
	}
	return opts
}

type ResourceQueryOption struct {
	resourceQuery string
}

func (r ResourceQueryOption) applyOnGet(opts getOptions) getOptions {
	if r.resourceQuery != "" {
		opts.opts = append(opts.opts, coap.WithQuery(r.resourceQuery))
	}
	return opts
}

func (r ResourceQueryOption) applyOnObserve(opts observeOptions) observeOptions {
	if r.resourceQuery != "" {
		opts.opts = append(opts.opts, coap.WithQuery(r.resourceQuery))
	}
	return opts
}

func (r ResourceQueryOption) applyOnUpdate(opts updateOptions) updateOptions {
	if r.resourceQuery != "" {
		opts.opts = append(opts.opts, coap.WithQuery(r.resourceQuery))
	}
	return opts
}

func (r ResourceQueryOption) applyOnCreate(opts createOptions) createOptions {
	if r.resourceQuery != "" {
		opts.opts = append(opts.opts, coap.WithQuery(r.resourceQuery))
	}
	return opts
}

func (r ResourceQueryOption) applyOnDelete(opts deleteOptions) deleteOptions {
	if r.resourceQuery != "" {
		opts.opts = append(opts.opts, coap.WithQuery(r.resourceQuery))
	}
	return opts
}

func (r ResourceQueryOption) applyOnCommonCommand(opts commonCommandOptions) commonCommandOptions {
	if r.resourceQuery != "" {
		opts.opts = append(opts.opts, coap.WithQuery(r.resourceQuery))
	}
	return opts
}

func (r ResourceQueryOption) applyOnGetDevice(opts getDeviceOptions) getDeviceOptions {
	if r.resourceQuery != "" {
		opts.opts = append(opts.opts, coap.WithQuery(r.resourceQuery))
	}
	return opts
}

func (r ResourceQueryOption) applyOnGetDeviceByIP(opts getDeviceByIPOptions) getDeviceByIPOptions {
	if r.resourceQuery != "" {
		opts.opts = append(opts.opts, coap.WithQuery(r.resourceQuery))
	}
	return opts
}

type DiscoveryConfigurationOption struct {
	cfg core.DiscoveryConfiguration
}

func (r DiscoveryConfigurationOption) applyOnGetDevice(opts getDeviceOptions) getDeviceOptions {
	opts.discoveryConfiguration = r.cfg
	return opts
}

func (r DiscoveryConfigurationOption) applyOnGetDevices(opts getDevicesOptions) getDevicesOptions {
	opts.discoveryConfiguration = r.cfg
	return opts
}

func (r DiscoveryConfigurationOption) applyOnGetGetDevicesWithHandler(opts getDevicesWithHandlerOptions) getDevicesWithHandlerOptions {
	opts.discoveryConfiguration = r.cfg
	return opts
}

func (r DiscoveryConfigurationOption) applyOnObserveDevices(opts observeDevicesOptions) observeDevicesOptions {
	opts.discoveryConfiguration = r.cfg
	return opts
}

func (r DiscoveryConfigurationOption) applyOnGet(opts getOptions) getOptions {
	opts.discoveryConfiguration = r.cfg
	return opts
}

func (r DiscoveryConfigurationOption) applyOnUpdate(opts updateOptions) updateOptions {
	opts.discoveryConfiguration = r.cfg
	return opts
}

func (r DiscoveryConfigurationOption) applyOnOwn(opts ownOptions) ownOptions {
	opts.discoveryConfiguration = r.cfg
	return opts
}

func (r DiscoveryConfigurationOption) applyOnCommonCommand(opts commonCommandOptions) commonCommandOptions {
	opts.discoveryConfiguration = r.cfg
	return opts
}

func (r DiscoveryConfigurationOption) applyOnObserve(opts observeOptions) observeOptions {
	opts.discoveryConfiguration = r.cfg
	return opts
}

func (r DiscoveryConfigurationOption) applyOnCreate(opts createOptions) createOptions {
	opts.discoveryConfiguration = r.cfg
	return opts
}

func (r DiscoveryConfigurationOption) applyOnDelete(opts deleteOptions) deleteOptions {
	opts.discoveryConfiguration = r.cfg
	return opts
}

// WithDiscoveryConfiguration allows to setup multicast request. By default it is send to ipv4 and ipv6.
func WithDiscoveryConfiguration(cfg core.DiscoveryConfiguration) DiscoveryConfigurationOption {
	return DiscoveryConfigurationOption{
		cfg: cfg,
	}
}

type LinkNotFoundCallback struct {
	linkNotFoundCallback func(links schema.ResourceLinks, href string) (schema.ResourceLink, error)
}

func (r LinkNotFoundCallback) applyOnGet(opts getOptions) getOptions {
	opts.linkNotFoundCallback = r.linkNotFoundCallback
	return opts
}

func (r LinkNotFoundCallback) applyOnUpdate(opts updateOptions) updateOptions {
	opts.linkNotFoundCallback = r.linkNotFoundCallback
	return opts
}

func (r LinkNotFoundCallback) applyOnObserve(opts observeOptions) observeOptions {
	opts.linkNotFoundCallback = r.linkNotFoundCallback
	return opts
}

func (r LinkNotFoundCallback) applyOnDelete(opts deleteOptions) deleteOptions {
	opts.linkNotFoundCallback = r.linkNotFoundCallback
	return opts
}

// WithLinkNotFoundCallback is used if the target of the link is not known e.g. the resource is not present on the device
// or the resource is non-discoverable.
// linkNotFoundCallback is a function which is called with links and href of the resource.
// It returns a link and an error. If the error is not nil, then the link is skipped.
// Otherwise the link is replaced with the returned link.
func WithLinkNotFoundCallback(linkNotFoundCallback func(links schema.ResourceLinks, href string) (schema.ResourceLink, error)) LinkNotFoundCallback {
	return LinkNotFoundCallback{
		linkNotFoundCallback: linkNotFoundCallback,
	}
}

// GetOption option definition.
type GetOption = interface {
	applyOnGet(opts getOptions) getOptions
}

type getOptions struct {
	opts                   []coap.OptionFunc
	codec                  coap.Codec
	discoveryConfiguration core.DiscoveryConfiguration
	linkNotFoundCallback   func(links schema.ResourceLinks, href string) (schema.ResourceLink, error)
}

type updateOptions struct {
	opts                   []coap.OptionFunc
	codec                  coap.Codec
	discoveryConfiguration core.DiscoveryConfiguration
	linkNotFoundCallback   func(links schema.ResourceLinks, href string) (schema.ResourceLink, error)
}

type createOptions struct {
	opts                   []coap.OptionFunc
	codec                  coap.Codec
	discoveryConfiguration core.DiscoveryConfiguration
}

type deleteOptions struct {
	opts                   []coap.OptionFunc
	codec                  coap.Codec
	discoveryConfiguration core.DiscoveryConfiguration
	linkNotFoundCallback   func(links schema.ResourceLinks, href string) (schema.ResourceLink, error)
}

// CreateOption option definition.
type CreateOption = interface {
	applyOnCreate(opts createOptions) createOptions
}

// DeleteOption option definition.
type DeleteOption = interface {
	applyOnDelete(opts deleteOptions) deleteOptions
}

// UpdateOption option definition.
type UpdateOption = interface {
	applyOnUpdate(opts updateOptions) updateOptions
}

// GetDevicesOption option definition.
type GetDevicesOption = interface {
	applyOnGetDevices(opts getDevicesOptions) getDevicesOptions
}

// GetDeviceOption option definition.
type GetDeviceOption = interface {
	applyOnGetDevice(opts getDeviceOptions) getDeviceOptions
}

// GetDeviceByIPOption option definition.
type GetDeviceByIPOption = interface {
	applyOnGetDeviceByIP(opts getDeviceByIPOptions) getDeviceByIPOptions
}

type GetDevicesWithHandlerOption = interface {
	applyOnGetGetDevicesWithHandler(opts getDevicesWithHandlerOptions) getDevicesWithHandlerOptions
}

type ObserveDevicesOption = interface {
	applyOnObserveDevices(opts observeDevicesOptions) observeDevicesOptions
}

type GetDetailsFunc = func(context.Context, *core.Device, schema.ResourceLinks, ...func(message.Options) message.Options) (interface{}, error)

type GetDetailsOption struct {
	getDetails GetDetailsFunc
}

func (r GetDetailsOption) applyOnGetDevices(opts getDevicesOptions) getDevicesOptions {
	opts.getDetails = r.getDetails
	return opts
}

func (r GetDetailsOption) applyOnGetDevice(opts getDeviceOptions) getDeviceOptions {
	opts.getDetails = r.getDetails
	return opts
}

func (r GetDetailsOption) applyOnGetDeviceByIP(opts getDeviceByIPOptions) getDeviceByIPOptions {
	opts.getDetails = r.getDetails
	return opts
}

type getDevicesOptions struct {
	resourceTypes          []string
	getDetails             GetDetailsFunc
	discoveryConfiguration core.DiscoveryConfiguration
	useDeviceID            bool
}

type getDeviceOptions struct {
	getDetails             GetDetailsFunc
	discoveryConfiguration core.DiscoveryConfiguration
	opts                   []coap.OptionFunc
}

type getDeviceByIPOptions struct {
	getDetails GetDetailsFunc
	opts       []coap.OptionFunc
}

type getDevicesWithHandlerOptions struct {
	discoveryConfiguration core.DiscoveryConfiguration
}

type observeDevicesOptions struct {
	discoveryConfiguration core.DiscoveryConfiguration
}

type ResourceTypesOption struct {
	resourceTypes []string
}

func (r ResourceTypesOption) applyOnGetDevices(opts getDevicesOptions) getDevicesOptions {
	opts.resourceTypes = r.resourceTypes
	return opts
}

func (r ResourceTypesOption) applyOnGet(opts getOptions) getOptions {
	for _, r := range r.resourceTypes {
		opts.opts = append(opts.opts, coap.WithResourceType(r))
	}
	return opts
}

type CodecOption struct {
	codec coap.Codec
}

func (r CodecOption) applyOnCreate(opts createOptions) createOptions {
	opts.codec = r.codec
	return opts
}

func (r CodecOption) applyOnDelete(opts deleteOptions) deleteOptions {
	opts.codec = r.codec
	return opts
}

func (r CodecOption) applyOnGet(opts getOptions) getOptions {
	opts.codec = r.codec
	return opts
}

func (r CodecOption) applyOnUpdate(opts updateOptions) updateOptions {
	opts.codec = r.codec
	return opts
}

func (r CodecOption) applyOnObserve(opts observeOptions) observeOptions {
	opts.codec = r.codec
	return opts
}

type observeOptions struct {
	codec                  coap.Codec
	opts                   []coap.OptionFunc
	resourceInterface      string
	discoveryConfiguration core.DiscoveryConfiguration
	linkNotFoundCallback   func(links schema.ResourceLinks, href string) (schema.ResourceLink, error)
}

// ObserveOption option definition.
type ObserveOption = interface {
	applyOnObserve(opts observeOptions) observeOptions
}

type OTMType int

const (
	OTMType_Manufacturer OTMType = 0
	OTMType_JustWorks    OTMType = 1
)

type ownOptions struct {
	opts                   []core.OwnOption
	otmTypes               []OTMType
	discoveryConfiguration core.DiscoveryConfiguration
}

// OwnOption option definition.
type OwnOption = interface {
	applyOnOwn(opts ownOptions) ownOptions
}

type presharedKeyOption struct {
	presharedKey []byte
}

func (r presharedKeyOption) applyOnOwn(opts ownOptions) ownOptions {
	opts.opts = append(opts.opts, core.WithPresharedKey(r.presharedKey))
	return opts
}

type actionDuringOwnOption struct {
	actionDuringOwn func(ctx context.Context, client *coap.ClientCloseHandler) (string, error)
}

func (r actionDuringOwnOption) applyOnOwn(opts ownOptions) ownOptions {
	opts.opts = append(opts.opts, core.WithActionDuringOwn(r.actionDuringOwn))
	return opts
}

type actionAfterOwnOption struct {
	actionAfterOwn func(ctx context.Context, client *coap.ClientCloseHandler) error
}

func (r actionAfterOwnOption) applyOnOwn(opts ownOptions) ownOptions {
	opts.opts = append(opts.opts, core.WithActionAfterOwn(r.actionAfterOwn))
	return opts
}

type otmOption struct {
	otmTypes []OTMType
}

func (r otmOption) applyOnOwn(opts ownOptions) ownOptions {
	opts.otmTypes = r.otmTypes
	return opts
}

type commonCommandOptions struct {
	discoveryConfiguration core.DiscoveryConfiguration
	opts                   []coap.OptionFunc
}

// CommonCommandOption option definition.
type CommonCommandOption = interface {
	applyOnCommonCommand(opts commonCommandOptions) commonCommandOptions
}

func applyCommonOptions(opts ...CommonCommandOption) commonCommandOptions {
	cfg := commonCommandOptions{
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnCommonCommand(cfg)
	}
	return cfg
}

func (r commonCommandOptions) applyOnGetDevice(opts getDeviceOptions) getDeviceOptions {
	opts.discoveryConfiguration = r.discoveryConfiguration
	return opts
}
