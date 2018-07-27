package ocfsdk

type OCFLimitI interface {
	ValidateValue(interface{}) error
}

type OCFBoolLimit struct {
}

func (a *OCFBoolLimit) ValidateValue(val interface{}) error {
	switch val.(type) {
	case bool:
		return nil
	default:
		return ErrInvalidType
	}
}

type OCFEnumLimit struct {
	ValidValues []string
}

func (a *OCFEnumLimit) ValidateValue(val interface{}) error {
	switch val.(type) {
	case string:
		for _, v := range a.ValidValues {
			if v == *val.(*string) {
				return nil
			}
		}
		return ErrInvalidEnumValue
	default:
		return ErrInvalidType
	}
}

type OCFIntLimit struct {
	Limit func(val int) error
}

func (a *OCFIntLimit) ValidateValue(val interface{}) error {
	switch val.(type) {
	case int:
		if a.Limit != nil {
			return a.Limit(*val.(*int))
		}
		return nil
	default:
		return ErrInvalidType
	}
}

type OCFStringLimit struct {
	Limit func(val *string) error
}

func (a *OCFStringLimit) ValidateValue(val interface{}) error {
	switch val.(type) {
	case string:
		if a.Limit != nil {
			return a.Limit(val.(*string))
		}
		return nil
	default:
		return ErrInvalidType
	}
}

type OCFDoubleLimit struct {
	Limit func(val float64) error
}

func (a *OCFDoubleLimit) ValidateValue(val interface{}) error {
	switch val.(type) {
	case float32:
		if a.Limit != nil {
			return a.Limit(*val.(*float64))
		}
		return nil
	case float64:
		if a.Limit != nil {
			return a.Limit(*val.(*float64))
		}
		return nil
	default:
		return ErrInvalidType
	}
}

type OCFByteLimit struct {
	Limit func(val []byte) error
}

func (a *OCFByteLimit) ValidateValue(val interface{}) error {
	switch val.(type) {
	case []byte:
		if a.Limit != nil {
			return a.Limit(val.([]byte))
		}
		return nil
	default:
		return ErrInvalidType
	}
}

type OCFMapLimit struct {
	MapLimit map[string]OCFLimitI
}

func (a *OCFMapLimit) ValidateValue(val interface{}) error {
	switch val.(type) {
	case map[string]interface{}:
		m := val.(map[string]interface{})
		for key, v := range m {
			if l, ok := a.MapLimit[key]; ok {
				if err := l.ValidateValue(v); err != nil {
					return err
				}
			} else {
				return ErrInvalidKeyOfMap
			}
		}
		return nil
	default:
		return ErrInvalidType
	}
}

type OCFArrayLimit struct {
	Limit OCFLimitI
}

func (a *OCFArrayLimit) ValidateValue(val interface{}) error {
	switch val.(type) {
	case []interface{}:
		m := val.([]interface{})
		for _, v := range m {
			if err := a.Limit.ValidateValue(v); err != nil {
				return err
			}
		}
		return nil
	default:
		return ErrInvalidType
	}
}
