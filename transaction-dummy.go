package ocfsdk

type transactionDummy struct {
}

func (t *transactionDummy) Commit() error {
	return nil
}

func (t *transactionDummy) Close() error {
	return nil
}
