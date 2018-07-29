package ocfsdk

type OCFAttributeIteratorI interface {
	Next() bool
	Value() OCFAttributeI
	Error() error
}

type OCFAttributeIterator struct {
	currentIdx int
	attributes []OCFAttributeI
	err        error
}

func (i *OCFAttributeIterator) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(i.attributes) {
		return true
	}
	return false
}

func (i *OCFAttributeIterator) Value() OCFAttributeI {
	if i.currentIdx < len(i.attributes) {
		return i.attributes[i.currentIdx]
	}
	i.err = ErrInvalidIterator
	return nil
}

func (i *OCFAttributeIterator) Error() error {
	return i.err
}

type OCFResourceTypeI interface {
	OCFIdI
	NewAttributeIterator() OCFAttributeIteratorI
}

type OCFResourceType struct {
	OCFId
	attributes []OCFAttributeI
}

func (rt *OCFResourceType) NewAttributeIterator() OCFAttributeIteratorI {
	return &OCFAttributeIterator{currentIdx: 0, attributes: rt.attributes}
}

func NewResourceType(id string, attributes []OCFAttributeI) (OCFResourceTypeI, error) {
	if len(id) == 0 || len(attributes) == 0 {
		return nil, ErrInvalidParams
	}
	return &OCFResourceType{OCFId: OCFId{id: id}, attributes: attributes}, nil
}
