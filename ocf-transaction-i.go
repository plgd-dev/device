package ocfsdk

type OCFTransactionI interface {
	Commit() error
	Drop() error
}

type OCFDummyTransaction struct {
}

func (t *OCFDummyTransaction) Commit() error {
	return nil
}

func (t *OCFDummyTransaction) Drop() error {
	return nil
}
