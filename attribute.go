package ocfsdk

type attribute struct {
	id
	value ValueI
	limit ValidatorI
}

func (a *attribute) GetValue(transaction TransactionI) (PayloadI, error) {
	if v, ok := a.value.(ValueGetI); ok {
		return v.Get(transaction)
	}

	return nil, ErrAccessDenied
}

func (a *attribute) SetValue(transaction TransactionI, value PayloadI) error {
	if err := a.limit.ValidateValue(value); err != nil {
		return err
	}
	if v, ok := a.value.(ValueSetI); ok {
		return v.Set(transaction, value)
	}

	return ErrAccessDenied
}

//NewAttribute creates attribute with name,value and limit
func NewAttribute(name string, value ValueI, limit ValidatorI) (AttributeI, error) {

	if len(name) == 0 || value == nil || limit == nil {
		return nil, ErrInvalidParams
	}
	return &attribute{id: id{id: name}, value: value, limit: limit}, nil
}
