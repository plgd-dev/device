package ocfsdk

type BoolROValue struct {
	get func(transaction TransactionI) (bool, error)
}

type BoolWOValue struct {
	set func(transaction TransactionI, s bool) (err error)
}

type BoolRWValue struct {
	BoolROValue
	BoolWOValue
}

func (v *BoolROValue) Get(transaction TransactionI) (bool, error) {
	if v.get != nil {
		return v.get(transaction)
	}
	return false, ErrOperationNotSupported
}

func (v *BoolROValue) GetValue(transaction TransactionI) (interface{}, error) {
	return v.Get(transaction)
}

func (v *BoolWOValue) Set(transaction TransactionI, s bool) (err error) {
	if v.set != nil {
		return v.set(transaction, s)
	}
	return ErrOperationNotSupported
}

func (v *BoolWOValue) SetValue(transaction TransactionI, s interface{}) error {
	return v.Set(transaction, s.(bool))
}

func NewBoolValue(get func(transaction TransactionI) (bool, error), set func(transaction TransactionI, s bool) error) (ValueI, error) {
	if get == nil && set == nil {
		return nil, ErrInvalidParams
	}
	if set == nil {
		return &BoolROValue{get: get}, nil
	}
	if get == nil {
		return &BoolWOValue{set: set}, nil
	}
	return &BoolRWValue{BoolROValue: BoolROValue{get: get}, BoolWOValue: BoolWOValue{set: set}}, nil
}

type MapROValue struct {
	get func(transaction TransactionI) (map[string]interface{}, error)
}

type MapWOValue struct {
	set func(transaction TransactionI, s map[string]interface{}) error
}

type MapRWValue struct {
	MapROValue
	MapWOValue
}

func (v *MapROValue) Get(transaction TransactionI) (map[string]interface{}, error) {
	if v.get != nil {
		return v.get(transaction)
	}
	return nil, ErrOperationNotSupported
}

func (v *MapROValue) GetValue(transaction TransactionI) (interface{}, error) {
	return v.Get(transaction)
}

func (v *MapWOValue) Set(transaction TransactionI, s map[string]interface{}) (err error) {
	if v.set != nil {
		return v.set(transaction, s)
	}
	return ErrOperationNotSupported
}

func (v *MapWOValue) SetValue(transaction TransactionI, s interface{}) error {
	return v.Set(transaction, s.(map[string]interface{}))
}

func NewMapValue(get func(transaction TransactionI) (map[string]interface{}, error), set func(transaction TransactionI, s map[string]interface{}) error) (ValueI, error) {
	if get == nil && set == nil {
		return nil, ErrInvalidParams
	}
	if set == nil {
		return &MapROValue{get: get}, nil
	}
	if get == nil {
		return &MapWOValue{set: set}, nil
	}
	return &MapRWValue{MapROValue: MapROValue{get: get}, MapWOValue: MapWOValue{set: set}}, nil
}
