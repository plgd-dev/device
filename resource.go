package ocfsdk

type resourceTypeIterator struct {
	MapIteratorMiddleware
}

func (i *resourceTypeIterator) Value() ResourceTypeI {
	v := i.ValueInterface()
	if v != nil {
		return v.(ResourceTypeI)
	}
	return nil
}

type resourceInterfaceIterator struct {
	MapIteratorMiddleware
}

func (i *resourceInterfaceIterator) Value() ResourceInterfaceI {
	v := i.ValueInterface()
	if v != nil {
		return v.(ResourceInterfaceI)
	}
	return nil
}

//ResourceParams parameters to initialize a resource
type ResourceParams struct {
	id                 string
	Discoverable       bool                 // true if resource are discoverable
	Observeable        bool                 // true if resource are observeable
	ResourceTypes      []ResourceTypeI      // list of resource types
	ResourceInterfaces []ResourceInterfaceI // list of interfaces types
	ResourceOperations ResourceOperationI   // actions that are supported by device
}

type resource struct {
	id
	discoverable bool
	observeable  bool

	resourceTypes      map[interface{}]interface{}
	resourceInterfaces map[interface{}]interface{}
	resourceOperations ResourceOperationI
}

func (r *resource) IsDiscoverable() bool {
	return r.discoverable
}

func (r *resource) IsObserveable() bool {
	return r.observeable
}

func (r *resource) NewResourceTypeIterator() ResourceTypeIteratorI {
	return &resourceTypeIterator{MapIteratorMiddleware: MapIteratorMiddleware{i: NewMapIterator(r.resourceTypes)}}
}

func (r *resource) NewResourceInterfaceIterator() ResourceInterfaceIteratorI {
	return &resourceInterfaceIterator{MapIteratorMiddleware: MapIteratorMiddleware{i: NewMapIterator(r.resourceInterfaces)}}
}

func (r *resource) GetResourceType(id string) (ResourceTypeI, error) {
	if v, ok := r.resourceTypes[id].(ResourceTypeI); ok {
		return v, nil
	}
	return nil, ErrNotExist
}

func (r *resource) GetResourceInterface(id string) (ResourceInterfaceI, error) {
	if v, ok := r.resourceInterfaces[id].(ResourceInterfaceI); ok {
		return v, nil
	}
	return nil, ErrNotExist
}

func (r *resource) GetResourceOperations() ResourceOperationI {
	return r.resourceOperations
}

//NewResource creates a resource by the params
func NewResource(params *ResourceParams) (ResourceI, error) {
	if len(params.id) == 0 || len(params.ResourceTypes) == 0 || params.ResourceOperations == nil {
		return nil, ErrInvalidParams
	}

	resourceInterfaces := make([]ResourceInterfaceI, 0)
	for _, val := range params.ResourceInterfaces {
		resourceInterfaces = append(resourceInterfaces, val)
	}

	haveDefaultIf := false
	haveBaselineIf := false

	for _, i := range resourceInterfaces {
		if i.GetID() == "" {
			haveDefaultIf = true
		}
		if i.GetID() == "oic.if.baseline" {
			haveBaselineIf = true
		}
		if haveDefaultIf && haveBaselineIf {
			break
		}
	}
	if !haveDefaultIf {
		resourceInterfaces = append(resourceInterfaces, NewResourceInterfaceBaseline(""))
	}
	if !haveBaselineIf {
		resourceInterfaces = append(resourceInterfaces, NewResourceInterfaceBaseline("oic.if.baseline"))
	}

	rt := make(map[interface{}]interface{})
	for _, val := range params.ResourceTypes {
		if rt[val.GetID()] != nil {
			return nil, ErrInvalidParams
		}
		rt[val.GetID()] = val
	}

	ifaces := make(map[interface{}]interface{})
	for _, val := range resourceInterfaces {
		if ifaces[val.GetID()] != nil {
			return nil, ErrInvalidParams
		}
		ifaces[val.GetID()] = val
	}

	return &resource{
		id:                 id{id: params.id},
		discoverable:       params.Discoverable,
		observeable:        params.Observeable,
		resourceTypes:      rt,
		resourceInterfaces: ifaces,
		resourceOperations: params.ResourceOperations,
	}, nil
}
