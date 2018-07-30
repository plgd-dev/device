package ocfsdk

type OCFBoolROValue struct {
	get func(transaction OCFTransactionI) (bool, error)
}

type OCFBoolWOValue struct {
	set func(transaction OCFTransactionI, s bool) (err error)
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

func (v *OCFBoolROValue) GetValue(transaction OCFTransactionI) (interface{}, error) {
	return v.Get(transaction)
}

func (v *OCFBoolWOValue) Set(transaction OCFTransactionI, s bool) (err error) {
	if v.set != nil {
		return v.set(transaction, s)
	}
	return ErrOperationNotSupported
}

func (v *OCFBoolWOValue) SetValue(transaction OCFTransactionI, s interface{}) error {
	return v.Set(transaction, s.(bool))
}

func NewBoolValue(get func(transaction OCFTransactionI) (bool, error), set func(transaction OCFTransactionI, s bool) error) (OCFValueI, error) {
	if get == nil && set == nil {
		return nil, ErrInvalidParams
	}
	if set == nil {
		return &OCFBoolROValue{get: get}, nil
	}
	if get == nil {
		return &OCFBoolWOValue{set: set}, nil
	}
	return &OCFBoolRWValue{OCFBoolROValue: OCFBoolROValue{get: get}, OCFBoolWOValue: OCFBoolWOValue{set: set}}, nil
}

type OCFMapROValue struct {
	get func(transaction OCFTransactionI) (map[string]interface{}, error)
}

type OCFMapWOValue struct {
	set func(transaction OCFTransactionI, s map[string]interface{}) error
}

type OCFMapRWValue struct {
	OCFMapROValue
	OCFMapWOValue
}

func (v *OCFMapROValue) Get(transaction OCFTransactionI) (map[string]interface{}, error) {
	if v.get != nil {
		return v.get(transaction)
	}
	return nil, ErrOperationNotSupported
}

func (v *OCFMapROValue) GetValue(transaction OCFTransactionI) (interface{}, error) {
	return v.Get(transaction)
}

func (v *OCFMapWOValue) Set(transaction OCFTransactionI, s map[string]interface{}) (err error) {
	if v.set != nil {
		return v.set(transaction, s)
	}
	return ErrOperationNotSupported
}

func (v *OCFMapWOValue) SetValue(transaction OCFTransactionI, s interface{}) error {
	return v.Set(transaction, s.(map[string]interface{}))
}

func NewMapValue(get func(transaction OCFTransactionI) (map[string]interface{}, error), set func(transaction OCFTransactionI, s map[string]interface{}) error) (OCFValueI, error) {
	if get == nil && set == nil {
		return nil, ErrInvalidParams
	}
	if set == nil {
		return &OCFMapROValue{get: get}, nil
	}
	if get == nil {
		return &OCFMapWOValue{set: set}, nil
	}
	return &OCFMapRWValue{OCFMapROValue: OCFMapROValue{get: get}, OCFMapWOValue: OCFMapWOValue{set: set}}, nil
}
