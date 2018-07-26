package main

type OCFAttributeI interface {
	IdI
	GetValue() (interface{}, error)
	SetValue(s interface{}) (bool, error)
}

type OCFAttribute struct {
	Id    IdI
	Value OCFValueI
	Limit OCFLimitI
}

func (a *OCFAttribute) GetValue() (interface{}, error) {
	switch v := a.Value.(type) {
	case OCFBoolValueGetI:
		return v.Get()
	case OCFEnumValueGetI:
		return v.Get()
	case OCFIntValueGetI:
		return v.Get()
	case OCFDoubleValueGetI:
		return v.Get()
	case OCFStringValueGetI:
		return v.Get()
	case OCFBinaryValueGetI:
		return v.Get()
		/*
			case OCFArrayValueGetI:
				return v.Get()
		*/
	case OCFMapValueGetI:
		return v.Get()
	}

	return nil, ErrAccessDenied
}

func (a *OCFAttribute) SetValue(s interface{}) (bool, error) {
	if err := a.Limit.ValidateValue(s); err != nil {
		return false, err
	}
	switch v := a.Value.(type) {
	case OCFBoolValueSetI:
		return v.Set(s.(bool))
	case OCFEnumValueSetI:
		return v.Set(s.(string))
	case OCFIntValueSetI:
		return v.Set(s.(int))
	case OCFDoubleValueSetI:
		return v.Set(s.(float64))
	case OCFStringValueSetI:
		return v.Set(s.(string))
	case OCFBinaryValueSetI:
		return v.Set(s.([]byte))
		/*
			case OCFArrayValueSetI:
				return v.Set(s.([]interface{}))
			case OCFMapValueSetI:
				return v.Set(s.(map[string]interface{}))
		*/
	}
	return false, ErrAccessDenied
}
