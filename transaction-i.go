package ocfsdk

//TransactionI defines interface for transaction over resource
type TransactionI interface {
	Commit() error //Commit used by update operation to store values to resource
	Close() error  //Close used by update/retrieve operation to clean values in transaction
}
