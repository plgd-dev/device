package ocfsdk

type OCFAttributeI interface {
	OCFIdI
	GetValue(transaction OCFTransactionI) (value interface{}, err error)
	SetValue(transaction OCFTransactionI, value interface{}) error
}

type OCFAttribute struct {
	OCFId
	Value OCFValueI
	Limit OCFLimitI
}

func (a *OCFAttribute) GetValue(transaction OCFTransactionI) (interface{}, error) {
	if v, ok := a.Value.(OCFValueGetI); ok {
		return v.GetValue(transaction)
	}

	return nil, ErrAccessDenied
}

func (a *OCFAttribute) SetValue(transaction OCFTransactionI, value interface{}) error {
	if err := a.Limit.ValidateValue(value); err != nil {
		return err
	}
	if v, ok := a.Value.(OCFValueSetI); ok {
		return v.SetValue(transaction, value)
	}

	return ErrAccessDenied
}

func NewAttribute(id string, value OCFValueI, limit OCFLimitI) (OCFAttributeI, error) {

	if len(id) == 0 || value == nil || limit == nil {
		return nil, ErrInvalidParams
	}
	return &OCFAttribute{OCFId: OCFId{id: id}, Value: value, Limit: limit}, nil
}
