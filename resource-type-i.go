package ocfsdk

//AttributeIteratorI defines interface of iterator over attributes
type AttributeIteratorI interface {
	MapIteratorI
	Value() AttributeI
}

//ResourceTypeI defines interface of resource type
type ResourceTypeI interface {
	IDI
	NewAttributeIterator() AttributeIteratorI
	GetAttribute(id string) (AttributeI, error)
}
