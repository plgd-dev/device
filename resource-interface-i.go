package ocfsdk

//ResourceInterfaceI defines interface of resource interface
type ResourceInterfaceI interface {
	IDI
}

//ResourceCreateInterfaceI defines interface of resource interface that is used by ResourceOperationCreateI
type ResourceCreateInterfaceI interface {
	ResourceInterfaceI
	Create(req RequestI, newResource ResourceI) (PayloadI, error)
}

//ResourceRetrieveInterfaceI defines interface of resource interface that is used by ResourceOperationRetrieveI
type ResourceRetrieveInterfaceI interface {
	ResourceInterfaceI
	Retrieve(req RequestI, trans TransactionI) (PayloadI, error)
}

//ResourceUpdateInterfaceI defines interface of resource interface that is used by ResourceOperationUpdateI
type ResourceUpdateInterfaceI interface {
	ResourceInterfaceI
	Update(req RequestI, trans TransactionI) (PayloadI, error)
}

//ResourceDeleteInterfaceI defines interface of resource interface that is used by ResourceOperationDeleteI
type ResourceDeleteInterfaceI interface {
	ResourceInterfaceI
	Delete(req RequestI, deletedResource ResourceI) (PayloadI, error)
}
