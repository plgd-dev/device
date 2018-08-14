package ocfsdk

import uuid "github.com/nu7hatch/gouuid"

//ResourceDeviceI defines interface of resource device
type ResourceDeviceI interface {
	ResourceI

	//Mandatory
	//GetDeviceName returns human friendly name defined by the vendor
	GetDeviceName() (string, error)
	//GetSpecVersion returns spec version of the core specification this device is implemented to
	GetSpecVersion() (string, error)
	//GetDeviceID returns unique identifier for Device.
	GetDeviceID() (*uuid.UUID, error)
	//GetDataModelVersion returns spec version of the Resource Specification to which this device datamodel is implemented
	GetDataModelVersion() (string, error)
	//GetProtocolIndependentID returns a unique and immutable Device identifier
	GetProtocolIndependentID() (*uuid.UUID, error)

	//Optional
	//GetManufacturerName  returns name of manufacturer of the Device, in one or more languages
	GetManufacturerName() ([]string, error)
	//GetModelNumber returns model number as designated by manufacturer
	GetModelNumber() (string, error)
}
