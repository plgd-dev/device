package ocfsdk

import (
	"reflect"
)

type ResourceTypeIterator struct {
	MapIterator
}

func (i *ResourceTypeIterator) Value() ResourceTypeI {
	v := i.value()
	if v != nil {
		return v.(ResourceTypeI)
	}
	return nil
}

type ResourceInterfaceIterator struct {
	MapIterator
}

func (i *ResourceInterfaceIterator) Value() ResourceInterfaceI {
	v := i.value()
	if v != nil {
		return v.(ResourceInterfaceI)
	}
	return nil
}

type ResourceParams struct {
	Id                 string
	Discoverable       bool
	Observeable        bool
	ResourceTypes      []ResourceTypeI
	ResourceInterfaces []ResourceInterfaceI
	ResourceOperations ResourceOperationI
}

type Resource struct {
	Id
	discoverable bool
	observeable  bool

	resourceTypes      map[interface{}]interface{}
	resourceInterfaces map[interface{}]interface{}
	resourceOperations ResourceOperationI
}

func (r *Resource) IsDiscoverable() bool {
	return r.discoverable
}

func (r *Resource) IsObserveable() bool {
	return r.observeable
}

func (r *Resource) NewResourceTypeIterator() ResourceTypeIteratorI {
	return &ResourceTypeIterator{MapIterator{data: r.resourceTypes, keys: reflect.ValueOf(r.resourceTypes).MapKeys(), currentIdx: 0, err: nil}}
}

func (r *Resource) NewResourceInterfaceIterator() ResourceInterfaceIteratorI {
	return &ResourceInterfaceIterator{MapIterator{data: r.resourceInterfaces, keys: reflect.ValueOf(r.resourceInterfaces).MapKeys(), currentIdx: 0, err: nil}}
}

func (r *Resource) GetResourceType(id string) (ResourceTypeI, error) {
	if v, ok := r.resourceTypes[id].(ResourceTypeI); ok {
		return v, nil
	}
	return nil, ErrNotExist
}

func (r *Resource) GetResourceInterface(id string) (ResourceInterfaceI, error) {
	if v, ok := r.resourceInterfaces[id].(ResourceInterfaceI); ok {
		return v, nil
	}
	return nil, ErrNotExist
}

func (r *Resource) GetResourceOperations() ResourceOperationI {
	return r.resourceOperations
}

func NewResource(params *ResourceParams) (ResourceI, error) {
	if len(params.Id) == 0 || len(params.ResourceTypes) == 0 || params.ResourceOperations == nil {
		return nil, ErrInvalidParams
	}

	resourceInterfaces := make([]ResourceInterfaceI, 0)
	for _, val := range params.ResourceInterfaces {
		resourceInterfaces = append(resourceInterfaces, val)
	}

	haveDefaultIf := false
	haveBaselineIf := false

	for _, i := range resourceInterfaces {
		if i.GetId() == "" {
			haveDefaultIf = true
		}
		if i.GetId() == "oic.if.baseline" {
			haveBaselineIf = true
		}
		if haveDefaultIf && haveBaselineIf {
			break
		}
	}
	if !haveDefaultIf {
		resourceInterfaces = append(resourceInterfaces, &ResourceInterfaceBaseline{ResourceInterface: ResourceInterface{Id: Id{id: ""}}})
	}
	if !haveBaselineIf {
		resourceInterfaces = append(resourceInterfaces, &ResourceInterfaceBaseline{ResourceInterface: ResourceInterface{Id: Id{id: "oic.if.baseline"}}})
	}

	rt := make(map[interface{}]interface{})
	for _, val := range params.ResourceTypes {
		if rt[val.GetId()] != nil {
			return nil, ErrInvalidParams
		}
		rt[val.GetId()] = val
	}

	ifaces := make(map[interface{}]interface{})
	for _, val := range resourceInterfaces {
		if ifaces[val.GetId()] != nil {
			return nil, ErrInvalidParams
		}
		ifaces[val.GetId()] = val
	}

	return &Resource{
		Id:                 Id{id: params.Id},
		discoverable:       params.Discoverable,
		observeable:        params.Observeable,
		resourceTypes:      rt,
		resourceInterfaces: ifaces,
		resourceOperations: params.ResourceOperations,
	}, nil
}
