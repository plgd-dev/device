package ocfsdk

//ResourceMiddleware defines middleware for resource
type ResourceMiddleware struct {
	resource ResourceI
}

//GetID returns id of object
func (rm *ResourceMiddleware) GetID() string {
	return rm.resource.GetID()
}

//IsDiscoverable returns true if resource are discoverable
func (rm *ResourceMiddleware) IsDiscoverable() bool {
	return rm.resource.IsDiscoverable()
}

//IsObserveable returns true if resource are observeable
func (rm *ResourceMiddleware) IsObserveable() bool {
	return rm.resource.IsObserveable()
}

//NewResourceTypeIterator get iretorar for iterate over resource types
func (rm *ResourceMiddleware) NewResourceTypeIterator() ResourceTypeIteratorI {
	return rm.resource.NewResourceTypeIterator()
}

//NewResourceInterfaceIterator get iretorar for iterate over resource interfaces
func (rm *ResourceMiddleware) NewResourceInterfaceIterator() ResourceInterfaceIteratorI {
	return rm.resource.NewResourceInterfaceIterator()
}

//GetResourceType returns a resource type by name
func (rm *ResourceMiddleware) GetResourceType(name string) (ResourceTypeI, error) {
	return rm.resource.GetResourceType(name)
}

//GetResourceInterface returns a resource interface by name
func (rm *ResourceMiddleware) GetResourceInterface(name string) (ResourceInterfaceI, error) {
	return rm.resource.GetResourceInterface(name)
}

//GetResourceOperations returns a resource operations
func (rm *ResourceMiddleware) GetResourceOperations() ResourceOperationI {
	return rm.resource.GetResourceOperations()
}
