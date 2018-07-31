package ocfsdk

type AttributeIteratorI interface {
	Next() bool
	Value() AttributeI
	Error() error
}

type AttributeIterator struct {
	currentIdx int
	attributes []AttributeI
	err        error
}

func (i *AttributeIterator) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(i.attributes) {
		return true
	}
	return false
}

func (i *AttributeIterator) Value() AttributeI {
	if i.currentIdx < len(i.attributes) {
		return i.attributes[i.currentIdx]
	}
	i.err = ErrInvalidIterator
	return nil
}

func (i *AttributeIterator) Error() error {
	return i.err
}

type ResourceTypeI interface {
	IdI
	NewAttributeIterator() AttributeIteratorI
}

type ResourceType struct {
	Id
	attributes []AttributeI
}

func (rt *ResourceType) NewAttributeIterator() AttributeIteratorI {
	return &AttributeIterator{currentIdx: 0, attributes: rt.attributes}
}

func NewResourceType(id string, attributes []AttributeI) (ResourceTypeI, error) {
	if len(id) == 0 || len(attributes) == 0 {
		return nil, ErrInvalidParams
	}
	return &ResourceType{Id: Id{id: id}, attributes: attributes}, nil
}
