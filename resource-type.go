package ocfsdk

type AttributeIteratorI interface {
	MapIteratorI
	Value() AttributeI
}

type AttributeIterator struct {
	MapIteratorMiddleware
}

func (i *AttributeIterator) Value() AttributeI {
	v := i.value()
	if v != nil {
		return v.(AttributeI)
	}
	return nil
}

type ResourceTypeI interface {
	IdI
	NewAttributeIterator() AttributeIteratorI
	GetAttribute(id string) (AttributeI, error)
}

type ResourceType struct {
	Id
	attributes map[interface{}]interface{}
}

func (rt *ResourceType) NewAttributeIterator() AttributeIteratorI {
	return &AttributeIterator{MapIteratorMiddleware: MapIteratorMiddleware{i: NewMapIterator(rt.attributes)}}
}

func (rt *ResourceType) GetAttribute(id string) (AttributeI, error) {
	if v, ok := rt.attributes[id].(AttributeI); ok {
		return v, nil
	}
	return nil, ErrNotExist
}

func NewResourceType(id string, attributes []AttributeI) (ResourceTypeI, error) {
	if len(id) == 0 {
		return nil, ErrInvalidParams
	}

	attr := make(map[interface{}]interface{})
	for _, val := range attributes {
		if attr[val.GetId()] != nil {
			return nil, ErrInvalidParams
		}
		attr[val.GetId()] = val
	}

	return &ResourceType{Id: Id{id: id}, attributes: attr}, nil
}
