package ocfsdk

//AttributeI defines interface for attribute
type AttributeI interface {
	IDI
	//GetValue returns simple value from attribute in transaction
	GetValue(transaction TransactionI) (value PayloadI, err error)
	//SetValue validate and set value to attribute in transaction
	SetValue(transaction TransactionI, value PayloadI) error
}
