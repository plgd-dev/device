package ocfsdk

//ResourceOperationI defines a base interface of operation over resource
type ResourceOperationI interface {
}

//ResourceOperationCreateI defines create interface of operation over resource
type ResourceOperationCreateI interface {
	ResourceOperationI
	Create(req RequestI) (PayloadI, error)
}

//ResourceOperationRetrieveI defines retrieve interface of operation over resource
type ResourceOperationRetrieveI interface {
	ResourceOperationI
	Retrieve(req RequestI) (PayloadI, error)
}

//ResourceOperationUpdateI defines update interface of operation over resource
type ResourceOperationUpdateI interface {
	ResourceOperationI
	Update(req RequestI) (PayloadI, error)
}

//ResourceOperationDeleteI defines delete interface of operation over resource
type ResourceOperationDeleteI interface {
	ResourceOperationI
	Delete(req RequestI) (PayloadI, error)
}
