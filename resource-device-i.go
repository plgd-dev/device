package ocfsdk

import uuid "github.com/nu7hatch/gouuid"

type ResourceDeviceI interface {
	ResourceI

	//Mandatory
	GetDeviceName() (string, error)
	GetSpecVersion() (string, error)
	GetDeviceId() (*uuid.UUID, error)
	GetDataModelVersion() (string, error)
	GetProtocolIndependentID() (*uuid.UUID, error)

	//Optional
	GetManufacturerName() ([]string, error)
	GetModelNumber() (string, error)
}
