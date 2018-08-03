package ocfsdk

import (
	"github.com/nu7hatch/gouuid"
)

type ResourceIteratorI interface {
	MapIteratorI
	Value() ResourceI
}

type DeviceI interface {
	AddResource(ResourceI) error
	DeleteResource(ResourceI) error
	NewResourceIterator() ResourceIteratorI
	GetDeviceId() (*uuid.UUID, error)
}
