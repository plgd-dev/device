package ocfsdk

type attributeIterator struct {
	MapIteratorMiddleware
}

func (i *attributeIterator) Value() AttributeI {
	v := i.ValueInterface()
	if v != nil {
		return v.(AttributeI)
	}
	return nil
}

type resourceType struct {
	id
	attributes map[interface{}]interface{}
}

func (rt *resourceType) NewAttributeIterator() AttributeIteratorI {
	return &attributeIterator{MapIteratorMiddleware: MapIteratorMiddleware{i: NewMapIterator(rt.attributes)}}
}

func (rt *resourceType) GetAttribute(id string) (AttributeI, error) {
	if v, ok := rt.attributes[id].(AttributeI); ok {
		return v, nil
	}
	return nil, ErrNotExist
}

//NewResourceType creates a resource type with the name and attributes
func NewResourceType(name string, attributes []AttributeI) (ResourceTypeI, error) {
	if len(name) == 0 {
		return nil, ErrInvalidParams
	}

	attr := make(map[interface{}]interface{})
	for _, val := range attributes {
		if attr[val.GetID()] != nil {
			return nil, ErrInvalidParams
		}
		attr[val.GetID()] = val
	}

	return &resourceType{id: id{id: name}, attributes: attr}, nil
}
