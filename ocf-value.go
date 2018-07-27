package ocfsdk

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

type OCFMapValue struct {
	get func(transaction OCFTransactionI) (map[string]OCFValueI, error)
}

func (v *OCFMapValue) Get(transaction OCFTransactionI) (map[string]OCFValueI, error) {
	if v.get != nil {
		return v.get(transaction)
	}
	return nil, ErrOperationNotSupported
}

func NewMapValue(get func(transaction OCFTransactionI) (map[string]OCFValueI, error)) (OCFValueI, error) {
	if get == nil {
		return nil, ErrInvalidParams
	}
	return &OCFMapValue{get: get}, nil
}
