package main

type OCFResourceTypeI interface {
	IdI
	GetAttributes() []OCFAttributeI
}

type OCFResourceType struct {
	Id         IdI
	Attributes []OCFAttributeI
}

func (rt *OCFResourceType) GetId() string {
	return rt.Id.GetId()
}

func (rt *OCFResourceType) GetAttributes() []OCFAttributeI {
	return rt.Attributes
}
