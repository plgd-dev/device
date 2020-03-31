package local

import (
	"context"

	kitNetCoap "github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/local/core"
	"github.com/go-ocf/sdk/schema"
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

func WithCodec(codec kitNetCoap.Codec) CodecOption {
	return CodecOption{
		codec: codec,
	}
}

func WithResourceTypes(resourceTypes ...string) ResourceTypesOption {
	return ResourceTypesOption{
		resourceTypes: resourceTypes,
	}
}

// WithIotivityHack set this option when device with iotivity 2.0 will be onboarded.
func WithIotivityHack() IotivityHackOption {
	return IotivityHackOption{}
}

type ResourceInterfaceOption struct {
	resourceInterface string
}

func (r ResourceInterfaceOption) applyOnGet(opts getOptions) getOptions {
	if r.resourceInterface != "" {
		opts.opts = append(opts.opts, kitNetCoap.WithInterface(r.resourceInterface))
	}
	return opts
}

func (r ResourceInterfaceOption) applyOnUpdate(opts updateOptions) updateOptions {
	if r.resourceInterface != "" {
		opts.opts = append(opts.opts, kitNetCoap.WithInterface(r.resourceInterface))
	}
	return opts
}

// GetOption option definition.
type GetOption = interface {
	applyOnGet(opts getOptions) getOptions
}

type getOptions struct {
	opts  []kitNetCoap.OptionFunc
	codec kitNetCoap.Codec
}

type updateOptions struct {
	opts  []kitNetCoap.OptionFunc
	codec kitNetCoap.Codec
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

type getDevicesOptions struct {
	resourceTypes []string
	err           func(error)
	getDetails    GetDetailsFunc
}

type getDeviceOptions struct {
	getDetails GetDetailsFunc
}

type ResourceTypesOption struct {
	resourceTypes []string
}

func (r ResourceTypesOption) applyOnGetDevices(opts getDevicesOptions) getDevicesOptions {
	opts.resourceTypes = r.resourceTypes
	return opts
}

type CodecOption struct {
	codec kitNetCoap.Codec
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
	codec kitNetCoap.Codec
}

// ObserveOption option definition.
type ObserveOption = interface {
	applyOnObserve(opts observeOptions) observeOptions
}

type ownOptions struct {
	opts []core.OwnOption
}

// OwnOption option definition.
type OwnOption = interface {
	applyOnOwn(opts ownOptions) ownOptions
}

type IotivityHackOption struct{}

func (r IotivityHackOption) applyOnOwn(opts ownOptions) ownOptions {
	opts.opts = append(opts.opts, core.WithIotivityHack())
	return opts
}
