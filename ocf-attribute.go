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
	switch v := a.Value.(type) {
	case OCFBoolValueGetI:
		return v.Get(transaction)
	case OCFEnumValueGetI:
		return v.Get(transaction)
	case OCFIntValueGetI:
		return v.Get(transaction)
	case OCFDoubleValueGetI:
		return v.Get(transaction)
	case OCFStringValueGetI:
		return v.Get(transaction)
	case OCFBinaryValueGetI:
		return v.Get(transaction)
		/*
			case OCFArrayValueGetI:
				return v.Get()
		*/
	case OCFMapValueGetI:
		return v.Get(transaction)
	}

	return nil, ErrAccessDenied
}

func (a *OCFAttribute) SetValue(transaction OCFTransactionI, value interface{}) error {
	if err := a.Limit.ValidateValue(value); err != nil {
		return err
	}
	switch v := a.Value.(type) {
	case OCFBoolValueSetI:
		return v.Set(transaction, value.(bool))
	case OCFEnumValueSetI:
		return v.Set(transaction, value.(string))
	case OCFIntValueSetI:
		return v.Set(transaction, value.(int))
	case OCFDoubleValueSetI:
		return v.Set(transaction, value.(float64))
	case OCFStringValueSetI:
		return v.Set(transaction, value.(string))
	case OCFBinaryValueSetI:
		return v.Set(transaction, value.([]byte))
		/*
			case OCFArrayValueSetI:
				return v.Set(s.([]interface{}))
			case OCFMapValueSetI:
				return v.Set(s.(map[string]interface{}))
		*/
	}
	return ErrAccessDenied
}

func NewAttribute(id string, value OCFValueI, limit OCFLimitI) (OCFAttributeI, error) {

	if len(id) == 0 || value == nil || limit == nil {
		return nil, ErrInvalidParams
	}
	return &OCFAttribute{OCFId: OCFId{id: id}, Value: value, Limit: limit}, nil
}
