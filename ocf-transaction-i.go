package ocfsdk

type OCFTransactionI interface {
	Commit() error
	Drop() error
}
