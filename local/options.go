package local

import kitNetCoap "github.com/go-ocf/kit/net/coap"

// WithInterface updates/gets resource with interface directly from a device.
func WithInterface(resourceInterface string) ResourceInterfaceOption {
	return ResourceInterfaceOption{
		resourceInterface: resourceInterface,
	}
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

// UpdateOption option definition.
type GetDevicesOption = interface {
	applyOnGetDevices(opts getDevicesOptions) getDevicesOptions
}

func WithError(err func(error)) ErrorOption {
	return ErrorOption{
		err: err,
	}
}

type ErrorOption struct {
	err func(error)
}

func (r ErrorOption) applyOnGetDevices(opts getDevicesOptions) getDevicesOptions {
	opts.err = r.err
	return opts
}

type getDevicesOptions struct {
	resourceTypes []string
	err           func(error)
}

func WithResourceTypes(resourceTypes ...string) ResourceTypesOption {
	return ResourceTypesOption{
		resourceTypes: resourceTypes,
	}
}

type ResourceTypesOption struct {
	resourceTypes []string
}

func (r ResourceTypesOption) applyOnGetDevices(opts getDevicesOptions) getDevicesOptions {
	opts.resourceTypes = r.resourceTypes
	return opts
}

func WithCodec(codec kitNetCoap.Codec) CodecOption {
	return CodecOption{
		codec: codec,
	}
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
