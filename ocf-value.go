package main

type OCFBoolROValue struct {
	get func() (bool, error)
}

type OCFBoolWOValue struct {
	setDefault func() error
	set        func(s bool) (changed bool, err error)
}

type OCFBoolRWValue struct {
	OCFBoolROValue
	OCFBoolWOValue
}

func (v *OCFBoolROValue) Get() (bool, error) {
	if v.get != nil {
		return v.get()
	}
	return false, ErrOperationNotSupported
}

func (v *OCFBoolWOValue) SetDefault() error {
	if v.setDefault != nil {
		return v.setDefault()
	}
	return ErrOperationNotSupported
}

func (v *OCFBoolWOValue) Set(s bool) (changed bool, err error) {
	if v.set != nil {
		return v.set(s)
	}
	return false, ErrOperationNotSupported
}

func NewBoolValue(get func() (bool, error), setDefault func() error, set func(s bool) (changed bool, err error)) (OCFValueI, error) {
	if get == nil && setDefault == nil && set == nil {
		return nil, ErrInvalidParams
	}
	if set == nil && setDefault == nil {
		return &OCFBoolROValue{get: get}, nil
	}
	if get == nil {
		return &OCFBoolWOValue{set: set, setDefault: setDefault}, nil
	}
	return &OCFBoolRWValue{OCFBoolROValue: OCFBoolROValue{get: get}, OCFBoolWOValue: OCFBoolWOValue{set: set, setDefault: setDefault}}, nil
}

type OCFMapValue struct {
	m map[string]OCFValueI
}

func (v *OCFMapValue) Get() (map[string]OCFValueI, error) {
	return v.m, nil
}

func NewMapValue(m map[string]OCFValueI) (OCFValueI, error) {
	return &OCFMapValue{m: m}, nil
}
