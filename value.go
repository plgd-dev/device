package ocfsdk

type ValueGet struct {
	get func(transaction TransactionI) (interface{}, error)
}

type ValueSet struct {
	set func(transaction TransactionI, s interface{}) (err error)
}

type ValueGetSet struct {
	ValueGet
	ValueSet
}

func (v *ValueGet) Get(transaction TransactionI) (interface{}, error) {
	if v.get != nil {
		return v.get(transaction)
	}
	return nil, ErrOperationNotSupported
}

func (v *ValueSet) Set(transaction TransactionI, s interface{}) error {
	if v.set != nil {
		return v.set(transaction, s)
	}
	return ErrOperationNotSupported
}

func NewValue(get func(transaction TransactionI) (interface{}, error), set func(transaction TransactionI, s interface{}) error) (ValueI, error) {
	if get == nil && set == nil {
		return nil, ErrInvalidParams
	}
	if set == nil {
		return &ValueGet{get: get}, nil
	}
	if get == nil {
		return &ValueSet{set: set}, nil
	}
	return &ValueGetSet{ValueGet: ValueGet{get: get}, ValueSet: ValueSet{set: set}}, nil
}
