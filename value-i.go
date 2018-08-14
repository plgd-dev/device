package ocfsdk

//ValueI defines a base interface of actions over value
type ValueI interface {
	//Set by type
	//Get by type
}

//ValueGetI defines interface to get a value
type ValueGetI interface {
	ValueI
	Get(transaction TransactionI) (PayloadI, error)
}

//ValueSetI defines interface to set a value
type ValueSetI interface {
	ValueI
	Set(transaction TransactionI, s PayloadI) error
}
