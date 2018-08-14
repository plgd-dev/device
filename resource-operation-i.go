package ocfsdk

type ResourceOperationI interface {
}

type ResourceOperationCreateI interface {
	Create(req RequestI) (PayloadI, error)
}

type ResourceOperationRetrieveI interface {
	Retrieve(req RequestI) (PayloadI, error)
}

type ResourceOperationUpdateI interface {
	Update(req RequestI) (PayloadI, error)
}

type ResourceOperationDeleteI interface {
	Delete(req RequestI) (PayloadI, error)
}
