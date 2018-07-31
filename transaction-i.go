package ocfsdk

type TransactionI interface {
	Commit() error
	Drop() error
}

type DummyTransaction struct {
}

func (t *DummyTransaction) Commit() error {
	return nil
}

func (t *DummyTransaction) Drop() error {
	return nil
}
