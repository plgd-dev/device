package ocfsdk

type PayloadI interface{}

type ResourceTypeIteratorI interface {
	MapIteratorI
	Value() ResourceTypeI
}

type ResourceInterfaceIteratorI interface {
	MapIteratorI
	Value() ResourceInterfaceI
}

type ResourceI interface {
	IdI

	IsDiscoverable() bool
	IsObserveable() bool
	NewResourceTypeIterator() ResourceTypeIteratorI
	NewResourceInterfaceIterator() ResourceInterfaceIteratorI
	GetResourceType(name string) (ResourceTypeI, error)
	GetResourceInterface(name string) (ResourceInterfaceI, error)
	GetResourceOperations() ResourceOperationI
}
