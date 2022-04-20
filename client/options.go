package client

import (
	"context"

	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
)

// WithInterface updates/gets resource with interface directly from a device.
func WithInterface(resourceInterface string) ResourceInterfaceOption {
	return ResourceInterfaceOption{
		resourceInterface: resourceInterface,
	}
}

func WithError(err func(error)) ErrorOption {
	return ErrorOption{
		err: err,
	}
}

func WithGetDetails(getDetails func(ctx context.Context, d *core.Device, links schema.ResourceLinks) (interface{}, error)) GetDetailsOption {
	return GetDetailsOption{
		getDetails: getDetails,
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

// WithActionDuringOwn allows to set deviceID of owned device and other staff over owner TLS.
// returns new deviceID, if it returns error device will be disowned.
func WithActionDuringOwn(actionDuringOwn func(ctx context.Context, client *coap.ClientCloseHandler) (string, error)) OwnOption {
	return actionDuringOwnOption{
		actionDuringOwn: actionDuringOwn,
	}
}

// WithActionAfterOwn allows initialize configuration at the device via DTLS connection with preshared key. For example setup time / NTP.
// if it returns error device will be disowned.
func WithActionAftersOwn(actionAfterOwn func(ctx context.Context, client *coap.ClientCloseHandler) error) OwnOption {
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

// WithOTM allows to set ownership transfer method, by default it is manufacturer.
func WithOTM(otmType OTMType) OwnOption {
	return otmOption{
		otmType: otmType,
	}
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

// GetOption option definition.
type GetOption = interface {
	applyOnGet(opts getOptions) getOptions
}

type getOptions struct {
	opts                   []coap.OptionFunc
	codec                  coap.Codec
	discoveryConfiguration core.DiscoveryConfiguration
}

type updateOptions struct {
	opts                   []coap.OptionFunc
	codec                  coap.Codec
	discoveryConfiguration core.DiscoveryConfiguration
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
}

// CreateOption option definition.
type CreateOption = interface {
	applyOnCreate(opts createOptions) createOptions
}

// UpdateOption option definition.
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

type ErrorOption struct {
	err func(error)
}

func (r ErrorOption) applyOnGetDevices(opts getDevicesOptions) getDevicesOptions {
	opts.err = r.err
	return opts
}

type GetDetailsFunc = func(context.Context, *core.Device, schema.ResourceLinks) (interface{}, error)

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
	err                    func(error)
	getDetails             GetDetailsFunc
	discoveryConfiguration core.DiscoveryConfiguration
}

type getDeviceOptions struct {
	getDetails             GetDetailsFunc
	discoveryConfiguration core.DiscoveryConfiguration
}

type getDeviceByIPOptions struct {
	getDetails GetDetailsFunc
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
	otmType                OTMType
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
	otmType OTMType
}

func (r otmOption) applyOnOwn(opts ownOptions) ownOptions {
	opts.otmType = r.otmType
	return opts
}

type commonCommandOptions struct {
	discoveryConfiguration core.DiscoveryConfiguration
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
