package ocfsdk

type ResourceMiddleware struct {
	resource ResourceI
}

func (rm *ResourceMiddleware) GetId() string {
	return rm.resource.GetId()
}

func (rm *ResourceMiddleware) IsDiscoverable() bool {
	return rm.resource.IsDiscoverable()
}

func (rm *ResourceMiddleware) IsObserveable() bool {
	return rm.resource.IsObserveable()
}

func (rm *ResourceMiddleware) NewResourceTypeIterator() ResourceTypeIteratorI {
	return rm.resource.NewResourceTypeIterator()
}

func (rm *ResourceMiddleware) NewResourceInterfaceIterator() ResourceInterfaceIteratorI {
	return rm.resource.NewResourceInterfaceIterator()
}

func (rm *ResourceMiddleware) GetResourceType(name string) (ResourceTypeI, error) {
	return rm.resource.GetResourceType(name)
}

func (rm *ResourceMiddleware) GetResourceInterface(name string) (ResourceInterfaceI, error) {
	return rm.resource.GetResourceInterface(name)
}

func (rm *ResourceMiddleware) GetResourceOperations() ResourceOperationI {
	return rm.resource.GetResourceOperations()
}
