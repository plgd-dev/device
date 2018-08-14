package ocfsdk

import (
	uuid "github.com/nu7hatch/gouuid"
)

const (
	deviceURI                                      = "/oic/d"
	deviceResourceType                             = "oic.wk.d"
	deviceResourceTypeAttributeName                = "n"
	deviceResourceTypeAttributeSpecVersion         = "icv"
	deviceResourceTypeAttributeID                  = "di"
	deviceResourceTypeAttributeDataModelVersion    = "dmv"
	deviceResourceTypeAttributeProtocolIndependent = "piid"
	deviceResourceTypeAttributeManufacturerName    = "dmn"
	deviceResourceTypeAttributeModelNumber         = "dmno"
)

//ResourceDeviceParams parameters to initialize resource device
type ResourceDeviceParams struct {
	DeviceName            string     // human friendly name defined by the vendor
	SpecVersion           string     // spec version of the core specification this device is implemented to
	DeviceID              *uuid.UUID // unique identifier for Device
	DataModelVersion      string     // spec version of the Resource Specification to which this device datamodel is implemented
	ProtocolIndependentID *uuid.UUID // a unique and immutable Device identifier
	ManufacturerName      []string   // name of manufacturer of the Device, in one or more languages
	ModelNumber           string     // model number as designated by manufacturer
}

type resourceDevice struct {
	ResourceMiddleware
}

func (d *resourceDevice) getAttributeValue(rtName, attrName string) (interface{}, error) {
	rt, err := d.GetResourceType(rtName)
	if err != nil {
		return nil, err
	}
	attr, err := rt.GetAttribute(attrName)
	if err != nil {
		return nil, err
	}
	val, err := attr.GetValue(nil)
	if err != nil {
		return nil, err
	}
	return val, nil
}

//Mandatory
func (d *resourceDevice) GetDeviceName() (string, error) {
	val, err := d.getAttributeValue(deviceResourceType, deviceResourceTypeAttributeName)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", ErrOperationNotSupported
}

func (d *resourceDevice) GetSpecVersion() (string, error) {
	val, err := d.getAttributeValue(deviceResourceType, deviceResourceTypeAttributeSpecVersion)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", ErrOperationNotSupported
}

func (d *resourceDevice) GetDeviceID() (*uuid.UUID, error) {
	val, err := d.getAttributeValue(deviceResourceType, deviceResourceTypeAttributeID)
	if err != nil {
		return nil, err
	}
	if str, ok := val.(string); ok {
		return uuid.ParseHex(str)
	}
	return nil, ErrOperationNotSupported
}

func (d *resourceDevice) GetDataModelVersion() (string, error) {
	val, err := d.getAttributeValue(deviceResourceType, deviceResourceTypeAttributeDataModelVersion)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", ErrOperationNotSupported
}

func (d *resourceDevice) GetProtocolIndependentID() (*uuid.UUID, error) {
	val, err := d.getAttributeValue(deviceResourceType, deviceResourceTypeAttributeProtocolIndependent)
	if err != nil {
		return nil, err
	}
	if str, ok := val.(string); ok {
		return uuid.ParseHex(str)
	}
	return nil, ErrOperationNotSupported
}

//Optional
func (d *resourceDevice) GetManufacturerName() ([]string, error) {
	val, err := d.getAttributeValue(deviceResourceType, deviceResourceTypeAttributeManufacturerName)
	if err != nil {
		return nil, err
	}
	if str, ok := val.([]string); ok {
		m := make([]string, len(str))
		for i, v := range str {
			m[i] = v
		}
		return m, nil
	}
	return nil, ErrOperationNotSupported
}

func (d *resourceDevice) GetModelNumber() (string, error) {
	val, err := d.getAttributeValue(deviceResourceType, deviceResourceTypeAttributeModelNumber)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", ErrOperationNotSupported
}

//NewResourceDevice creates a resource device by params
func NewResourceDevice(params *ResourceDeviceParams) (ResourceDeviceI, error) {
	if params.DeviceID == nil || params.ProtocolIndependentID == nil ||
		len(params.DeviceName) == 0 ||
		len(params.SpecVersion) == 0 ||
		len(params.DataModelVersion) == 0 {
		return nil, ErrInvalidParams
	}

	nVal, err := NewValue(func(TransactionI) (PayloadI, error) { return params.DeviceName, nil }, nil)
	if err != nil {
		return nil, err
	}

	n, err := NewAttribute(deviceResourceTypeAttributeName, nVal, &StringValidator{})
	if err != nil {
		return nil, err
	}

	icvVal, err := NewValue(func(TransactionI) (PayloadI, error) { return params.SpecVersion, nil }, nil)
	if err != nil {
		return nil, err
	}

	icv, err := NewAttribute(deviceResourceTypeAttributeSpecVersion, icvVal, &StringValidator{})
	if err != nil {
		return nil, err
	}

	diUUID := params.DeviceID.String()
	diVal, err := NewValue(func(TransactionI) (PayloadI, error) { return diUUID, nil }, nil)
	if err != nil {
		return nil, err
	}

	di, err := NewAttribute(deviceResourceTypeAttributeID, diVal, &StringValidator{})
	if err != nil {
		return nil, err
	}

	dmvVal, err := NewValue(func(TransactionI) (PayloadI, error) { return params.DataModelVersion, nil }, nil)
	if err != nil {
		return nil, err
	}

	dmv, err := NewAttribute(deviceResourceTypeAttributeDataModelVersion, dmvVal, &StringValidator{})
	if err != nil {
		return nil, err
	}

	piidUUID := params.ProtocolIndependentID.String()
	piidVal, err := NewValue(func(TransactionI) (PayloadI, error) { return piidUUID, nil }, nil)
	if err != nil {
		return nil, err
	}

	piid, err := NewAttribute(deviceResourceTypeAttributeProtocolIndependent, piidVal, &StringValidator{})
	if err != nil {
		return nil, err
	}

	attributes := []AttributeI{
		n,
		icv,
		di,
		dmv,
		piid,
	}

	if len(params.ModelNumber) > 0 {
		dmnoVal, err := NewValue(func(TransactionI) (PayloadI, error) { return params.ModelNumber, nil }, nil)
		if err != nil {
			return nil, err
		}

		dmno, err := NewAttribute(deviceResourceTypeAttributeModelNumber, dmnoVal, &StringValidator{})
		if err != nil {
			return nil, err
		}
		attributes = append(attributes, dmno)
	}

	if len(params.ManufacturerName) > 0 {
		dmnVal, err := NewValue(func(TransactionI) (PayloadI, error) { return params.ManufacturerName, nil }, nil)
		if err != nil {
			return nil, err
		}

		dmn, err := NewAttribute(deviceResourceTypeAttributeManufacturerName, dmnVal, &StringValidator{})
		if err != nil {
			return nil, err
		}
		attributes = append(attributes, dmn)
	}

	rt, err := NewResourceType(deviceResourceType, attributes)
	if err != nil {
		return nil, err
	}

	resourceParams := &ResourceParams{
		id:                 deviceURI,
		Discoverable:       true,
		Observeable:        true,
		ResourceTypes:      []ResourceTypeI{rt},
		ResourceOperations: NewResourceOperationRetrieve(func() (TransactionI, error) { return &transactionDummy{}, nil }),
	}

	resMid, err := NewResource(resourceParams)
	if err != nil {
		return nil, err
	}

	return &resourceDevice{ResourceMiddleware: ResourceMiddleware{resource: resMid}}, nil
}
