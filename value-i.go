package ocfsdk

type ValueI interface {
	//Set by type
	//Get by type
}

type ValueGetI interface {
	Get(transaction TransactionI) (interface{}, error)
}

type ValueSetI interface {
	Set(transaction TransactionI, s interface{}) error
}
