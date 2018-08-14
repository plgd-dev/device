package ocfsdk

//ResourceTypeIteratorI defines interface of iterator over resource types
type ResourceTypeIteratorI interface {
	MapIteratorI
	//Value returns resource type from iterator
	Value() ResourceTypeI
}

//ResourceInterfaceIteratorI defines interface of iterator over resource interfaces
type ResourceInterfaceIteratorI interface {
	MapIteratorI
	//Value returns resource interface from iterator
	Value() ResourceInterfaceI
}

//ResourceI defines interface of resource
type ResourceI interface {
	IDI

	//IsDiscoverable returns true if resource are discoverable
	IsDiscoverable() bool
	//IsObserveable returns true if resource are observeable
	IsObserveable() bool
	//NewResourceTypeIterator get iretorar for iterate over resource types
	NewResourceTypeIterator() ResourceTypeIteratorI
	//NewResourceInterfaceIterator get iretorar for iterate over resource interfaces
	NewResourceInterfaceIterator() ResourceInterfaceIteratorI
	//GetResourceType returns a resource type by name
	GetResourceType(name string) (ResourceTypeI, error)
	//GetResourceInterface returns a resource interface by name
	GetResourceInterface(name string) (ResourceInterfaceI, error)
	//GetResourceOperations returns a resource operations
	GetResourceOperations() ResourceOperationI
}
