package ocfsdk

//BoolValidator bool validator
type BoolValidator struct {
}

//ValidateValue validate of type value
func (a *BoolValidator) ValidateValue(val PayloadI) error {
	switch val.(type) {
	case bool:
		return nil
	default:
		return ErrInvalidType
	}
}

//EnumValidator enum validator
type EnumValidator struct {
	ValidValues []string
}

//ValidateValue validate of type value
func (a *EnumValidator) ValidateValue(val PayloadI) error {
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

//IntValidator int validator
type IntValidator struct {
}

//ValidateValue validate of type value
func (a *IntValidator) ValidateValue(val PayloadI) error {
	switch val.(type) {
	case int:
		return nil
	default:
		return ErrInvalidType
	}
}

//StringValidator string validator
type StringValidator struct {
}

//ValidateValue validate of type value
func (a *StringValidator) ValidateValue(val PayloadI) error {
	switch val.(type) {
	case string:
		return nil
	default:
		return ErrInvalidType
	}
}

//FloatValidator float validator
type FloatValidator struct{}

//ValidateValue validate of type value
func (a *FloatValidator) ValidateValue(val PayloadI) error {
	switch val.(type) {
	case float32:
		return nil
	case float64:
		return nil
	default:
		return ErrInvalidType
	}
}

//BytesValidator bytes validator
type BytesValidator struct {
}

//ValidateValue validate of type value
func (a *BytesValidator) ValidateValue(val PayloadI) error {
	switch val.(type) {
	case []byte:
		return nil
	default:
		return ErrInvalidType
	}
}

//MapValidator map validator
type MapValidator struct {
	ElementValidator map[interface{}]ValidatorI
}

//ValidateValue validate each value over map with ElementValidator
func (a *MapValidator) ValidateValue(val PayloadI) error {
	switch val.(type) {
	case map[interface{}]interface{}:
		m := val.(map[interface{}]interface{})
		for key, validator := range a.ElementValidator {
			if l, ok := m[key]; ok {
				if err := validator.ValidateValue(l); err != nil {
					return err
				}
			} else {
				return ErrUnprovidedKeyOfMap
			}
		}
		return nil
	default:
		return ErrInvalidType
	}
}

//ArrayValidator array validator
type ArrayValidator struct {
	ElementValidator ValidatorI
}

//ValidateValue validate of type value
func (a *ArrayValidator) ValidateValue(val PayloadI) error {
	switch val.(type) {
	case []PayloadI:
		m := val.([]PayloadI)
		for _, v := range m {
			if err := a.ElementValidator.ValidateValue(v); err != nil {
				return err
			}
		}
		return nil
	default:
		return ErrInvalidType
	}
}
