package ocfsdk

import (
	"github.com/nu7hatch/gouuid"
)

//ResourceIteratorI defines interface of iterator over resources
type ResourceIteratorI interface {
	MapIteratorI
	//Value returns resource from iterator
	Value() ResourceI
}

//DeviceI defines interface of device
type DeviceI interface {
	//AddResource add resource to the device
	AddResource(ResourceI) error
	//DeleteResource remove resource to the device
	DeleteResource(ResourceI) error
	//NewResourceIterator get iretorar for iterate over resources
	NewResourceIterator() ResourceIteratorI
	//GetDeviceID returns uuid of device
	GetDeviceID() (*uuid.UUID, error)
}
