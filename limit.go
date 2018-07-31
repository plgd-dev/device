package ocfsdk

type LimitI interface {
	ValidateValue(interface{}) error
}

type BoolLimit struct {
}

func (a *BoolLimit) ValidateValue(val interface{}) error {
	switch val.(type) {
	case bool:
		return nil
	default:
		return ErrInvalidType
	}
}

type EnumLimit struct {
	ValidValues []string
}

func (a *EnumLimit) ValidateValue(val interface{}) error {
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

type IntLimit struct {
	Limit func(val int) error
}

func (a *IntLimit) ValidateValue(val interface{}) error {
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

type StringLimit struct {
	Limit func(val *string) error
}

func (a *StringLimit) ValidateValue(val interface{}) error {
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

type DoubleLimit struct {
	Limit func(val float64) error
}

func (a *DoubleLimit) ValidateValue(val interface{}) error {
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

type ByteLimit struct {
	Limit func(val []byte) error
}

func (a *ByteLimit) ValidateValue(val interface{}) error {
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

type MapLimit struct {
	MapLimit map[string]LimitI
}

func (a *MapLimit) ValidateValue(val interface{}) error {
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

type ArrayLimit struct {
	Limit LimitI
}

func (a *ArrayLimit) ValidateValue(val interface{}) error {
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
