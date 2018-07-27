package ocfsdk

type OCFResourceTypeI interface {
	OCFIdI
	GetAttributes() []OCFAttributeI
}

type OCFResourceType struct {
	OCFId
	Attributes []OCFAttributeI
}

func (rt *OCFResourceType) GetAttributes() []OCFAttributeI {
	return rt.Attributes
}

func NewResourceType(id string, attributes []OCFAttributeI) (OCFResourceTypeI, error) {
	if len(id) == 0 || len(attributes) == 0 {
		return nil, ErrInvalidParams
	}
	return &OCFResourceType{OCFId: OCFId{Id: id}, Attributes: attributes}, nil
}
