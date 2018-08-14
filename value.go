package ocfsdk

type valueGet struct {
	get func(transaction TransactionI) (PayloadI, error)
}

type valueSet struct {
	set func(transaction TransactionI, s PayloadI) (err error)
}

func (v *valueGet) Get(transaction TransactionI) (PayloadI, error) {
	if v.get != nil {
		return v.get(transaction)
	}
	return nil, ErrOperationNotSupported
}

func (v *valueSet) Set(transaction TransactionI, s PayloadI) error {
	if v.set != nil {
		return v.set(transaction, s)
	}
	return ErrOperationNotSupported
}

//NewValue creates a value for attribute
func NewValue(get func(transaction TransactionI) (PayloadI, error), set func(transaction TransactionI, s PayloadI) error) (ValueI, error) {
	if get == nil && set == nil {
		return nil, ErrInvalidParams
	}
	if set == nil {
		return &valueGet{get: get}, nil
	}
	if get == nil {
		return &valueSet{set: set}, nil
	}

	type valueGetSet struct {
		valueGet
		valueSet
	}

	return &valueGetSet{valueGet: valueGet{get: get}, valueSet: valueSet{set: set}}, nil
}
