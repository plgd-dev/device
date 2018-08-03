package ocfsdk

import (
	uuid "github.com/nu7hatch/gouuid"
)

const (
	DEVICE_URI                            = "/oic/d"
	DEVICE_RESOURCE_TYPE                  = "oic.wk.d"
	DEVICE_ATTRIBUTE_NAME                 = "n"
	DEVICE_ATTRIBUTE_SPEC_VERSION         = "icv"
	DEVICE_ATTRIBUTE_ID                   = "di"
	DEVICE_ATTRIBUTE_DATA_MODEL_VERSION   = "dmv"
	DEVICE_ATTRIBUTE_PROTOCOL_INDEPENDENT = "piid"
	DEVICE_ATTRIBUTE_MANUFACTURER_NAME    = "dmn"
	DEVICE_ATTRIBUTE_MODEL_NUMBER         = "dmno"
)

type ResourceDeviceParams struct {
	DeviceName            string
	SpecVersion           string
	DeviceId              *uuid.UUID
	DataModelVersion      string
	ProtocolIndependentID *uuid.UUID
	ManufacturerName      []string
	ModelNumber           string
}

type ResourceDevice struct {
	ResourceMiddleware
}

func (d *ResourceDevice) getAttributeValue(rtName, attrName string) (interface{}, error) {
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
func (d *ResourceDevice) GetDeviceName() (string, error) {
	val, err := d.getAttributeValue(DEVICE_RESOURCE_TYPE, DEVICE_ATTRIBUTE_NAME)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", ErrOperationNotSupported
}

func (d *ResourceDevice) GetSpecVersion() (string, error) {
	val, err := d.getAttributeValue(DEVICE_RESOURCE_TYPE, DEVICE_ATTRIBUTE_SPEC_VERSION)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", ErrOperationNotSupported
}

func (d *ResourceDevice) GetDeviceId() (*uuid.UUID, error) {
	val, err := d.getAttributeValue(DEVICE_RESOURCE_TYPE, DEVICE_ATTRIBUTE_ID)
	if err != nil {
		return nil, err
	}
	if str, ok := val.(string); ok {
		return uuid.ParseHex(str)
	}
	return nil, ErrOperationNotSupported
}

func (d *ResourceDevice) GetDataModelVersion() (string, error) {
	val, err := d.getAttributeValue(DEVICE_RESOURCE_TYPE, DEVICE_ATTRIBUTE_DATA_MODEL_VERSION)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", ErrOperationNotSupported
}

func (d *ResourceDevice) GetProtocolIndependentID() (*uuid.UUID, error) {
	val, err := d.getAttributeValue(DEVICE_RESOURCE_TYPE, DEVICE_ATTRIBUTE_PROTOCOL_INDEPENDENT)
	if err != nil {
		return nil, err
	}
	if str, ok := val.(string); ok {
		return uuid.ParseHex(str)
	}
	return nil, ErrOperationNotSupported
}

//Optional
func (d *ResourceDevice) GetManufacturerName() ([]string, error) {
	val, err := d.getAttributeValue(DEVICE_RESOURCE_TYPE, DEVICE_ATTRIBUTE_MANUFACTURER_NAME)
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

func (d *ResourceDevice) GetModelNumber() (string, error) {
	val, err := d.getAttributeValue(DEVICE_RESOURCE_TYPE, DEVICE_ATTRIBUTE_MODEL_NUMBER)
	if err != nil {
		return "", err
	}
	if str, ok := val.(string); ok {
		return str, nil
	}
	return "", ErrOperationNotSupported
}

func NewResourceDevice(params *ResourceDeviceParams) (ResourceDeviceI, error) {
	if params.DeviceId == nil || params.ProtocolIndependentID == nil ||
		len(params.DeviceName) == 0 ||
		len(params.SpecVersion) == 0 ||
		len(params.DataModelVersion) == 0 {
		return nil, ErrInvalidParams
	}

	nVal, err := NewValue(func(TransactionI) (interface{}, error) { return params.DeviceName, nil }, nil)
	if err != nil {
		return nil, err
	}

	n, err := NewAttribute(DEVICE_ATTRIBUTE_NAME, nVal, &StringLimit{})
	if err != nil {
		return nil, err
	}

	icvVal, err := NewValue(func(TransactionI) (interface{}, error) { return params.SpecVersion, nil }, nil)
	if err != nil {
		return nil, err
	}

	icv, err := NewAttribute(DEVICE_ATTRIBUTE_SPEC_VERSION, icvVal, &StringLimit{})
	if err != nil {
		return nil, err
	}

	diUUID := params.DeviceId.String()
	diVal, err := NewValue(func(TransactionI) (interface{}, error) { return diUUID, nil }, nil)
	if err != nil {
		return nil, err
	}

	di, err := NewAttribute(DEVICE_ATTRIBUTE_ID, diVal, &StringLimit{})
	if err != nil {
		return nil, err
	}

	dmvVal, err := NewValue(func(TransactionI) (interface{}, error) { return params.DataModelVersion, nil }, nil)
	if err != nil {
		return nil, err
	}

	dmv, err := NewAttribute(DEVICE_ATTRIBUTE_DATA_MODEL_VERSION, dmvVal, &StringLimit{})
	if err != nil {
		return nil, err
	}

	piidUUID := params.ProtocolIndependentID.String()
	piidVal, err := NewValue(func(TransactionI) (interface{}, error) { return piidUUID, nil }, nil)
	if err != nil {
		return nil, err
	}

	piid, err := NewAttribute(DEVICE_ATTRIBUTE_PROTOCOL_INDEPENDENT, piidVal, &StringLimit{})
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
		dmnoVal, err := NewValue(func(TransactionI) (interface{}, error) { return params.ModelNumber, nil }, nil)
		if err != nil {
			return nil, err
		}

		dmno, err := NewAttribute(DEVICE_ATTRIBUTE_MODEL_NUMBER, dmnoVal, &StringLimit{})
		if err != nil {
			return nil, err
		}
		attributes = append(attributes, dmno)
	}

	if len(params.ManufacturerName) > 0 {
		dmnVal, err := NewValue(func(TransactionI) (interface{}, error) { return params.ManufacturerName, nil }, nil)
		if err != nil {
			return nil, err
		}

		dmn, err := NewAttribute(DEVICE_ATTRIBUTE_MANUFACTURER_NAME, dmnVal, &StringLimit{})
		if err != nil {
			return nil, err
		}
		attributes = append(attributes, dmn)
	}

	rt, err := NewResourceType(DEVICE_RESOURCE_TYPE, attributes)
	if err != nil {
		return nil, err
	}

	resourceParams := &ResourceParams{
		Id:                 DEVICE_URI,
		Discoverable:       true,
		Observeable:        true,
		ResourceTypes:      []ResourceTypeI{rt},
		ResourceOperations: NewResourceOperationRetrieve(func() (TransactionI, error) { return &DummyTransaction{}, nil }),
	}

	resMid, err := NewResource(resourceParams)
	if err != nil {
		return nil, err
	}

	return &ResourceDevice{ResourceMiddleware: ResourceMiddleware{resource: resMid}}, nil
}
