package ocfsdk

import "reflect"

type OCFBoolROValue struct {
	get func(transaction OCFTransactionI) (bool, error)
}

type OCFBoolWOValue struct {
	setDefault func(transaction OCFTransactionI) error
	set        func(transaction OCFTransactionI, s bool) (err error)
}

type OCFBoolRWValue struct {
	OCFBoolROValue
	OCFBoolWOValue
}

func (v *OCFBoolROValue) Get(transaction OCFTransactionI) (bool, error) {
	if v.get != nil {
		return v.get(transaction)
	}
	return false, ErrOperationNotSupported
}

func (v *OCFBoolWOValue) SetDefault(transaction OCFTransactionI) error {
	if v.setDefault != nil {
		return v.setDefault(transaction)
	}
	return ErrOperationNotSupported
}

func (v *OCFBoolWOValue) Set(transaction OCFTransactionI, s bool) (err error) {
	if v.set != nil {
		return v.set(transaction, s)
	}
	return ErrOperationNotSupported
}

func NewBoolValue(get func(transaction OCFTransactionI) (bool, error), setDefault func(transaction OCFTransactionI) error, set func(transaction OCFTransactionI, s bool) error) (OCFValueI, error) {
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

type OCFMapValueIterator struct {
	currentIdx int
	values     map[string]OCFValueI
	err        error
}

func (i *OCFMapValueIterator) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(reflect.ValueOf(i.values).MapKeys()) {
		return true
	}
	i.err = ErrInvalidIterator
	return false
}

func (i *OCFMapValueIterator) Key() string {
	if i.currentIdx < len(reflect.ValueOf(i.values).MapKeys()) {
		return reflect.ValueOf(i.values).MapKeys()[i.currentIdx].String()
	}
	i.err = ErrInvalidIterator
	return ""
}

func (i *OCFMapValueIterator) Error() error {
	return i.err
}

func (i *OCFMapValueIterator) Value() OCFValueI {
	if i.currentIdx < len(reflect.ValueOf(i.values).MapKeys()) {
		keys := reflect.ValueOf(i.values).MapKeys()
		return i.values[keys[i.currentIdx].String()]
	}
	i.err = ErrInvalidIterator
	return nil
}

type OCFMapValue struct {
	values map[string]OCFValueI
}

func (v *OCFMapValue) NewMapValueIterator() OCFMapValueIteratorI {
	return &OCFMapValueIterator{currentIdx: 0, values: v.values}
}

type OCFMapROValue struct {
	OCFMapValue
}

func (v *OCFMapROValue) Get(transaction OCFTransactionI) (ret map[string]interface{}, err error) {
	ret = make(map[string]interface{})
	for key, val := range v.values {
		if ret[key], err = val.(OCFValueGetI).GetValue(transaction); ret != nil {
			return nil, err
		}
	}
	return ret, nil
}

type OCFMapWOValue struct {
	OCFMapValue
}

func (v *OCFMapWOValue) SetDefault(transaction OCFTransactionI) error {
	for _, val := range v.values {
		if err := val.(OCFValueSetI).SetDefault(transaction); err != nil {
			return err
		}
	}
	return ErrOperationNotSupported
}

func (v *OCFMapWOValue) Set(transaction OCFTransactionI, s map[string]interface{}) (err error) {
	for key, val := range v.values {
		if setVal, ok := s[key]; ok {
			if err = val.(OCFValueSetI).SetValue(transaction, setVal); err != nil {
				return err
			}
		} else {
			if err = val.(OCFValueSetI).SetDefault(transaction); err != nil {
				return err
			}
		}

	}
	return nil
}

type OCFMapRWValue struct {
	OCFMapROValue
	OCFMapWOValue
}

/*
func NewMapValue(values map[string]OCFValueI) OCFValueI {
	return &OCFMapRWValue{OCFMapValue: OCFMapValue{values: values}}
}
*/
