package ocfsdk

//ValidatorI defines interface for validation of value
type ValidatorI interface {
	//ValidateValue of type value
	ValidateValue(value PayloadI) error
}
