package ocfsdk

import (
	"reflect"
	"sync"

	uuid "github.com/nu7hatch/gouuid"
)

type ResourceIterator struct {
	MapIterator
}

func (i *ResourceIterator) Value() ResourceI {
	v := i.value()
	if v != nil {
		return v.(ResourceI)
	}
	return nil
}

type Device struct {
	resources      map[interface{}]interface{}
	resourcesMutex sync.Mutex
}

func (d *Device) AddResource(r ResourceI) error {
	if r == nil {
		return ErrInvalidParams
	}
	d.resourcesMutex.Lock()
	defer d.resourcesMutex.Unlock()
	if d.resources[r.GetId()] != nil {
		return ErrExist
	}
	d.resources[r.GetId()] = r
	return nil
}

func (d *Device) DeleteResource(r ResourceI) error {
	if r == nil {
		return ErrInvalidParams
	}
	d.resourcesMutex.Lock()
	defer d.resourcesMutex.Unlock()
	if d.resources[r.GetId()] == nil {
		return ErrNotExist
	}
	delete(d.resources, r.GetId())
	return nil
}

func (d *Device) NewResourceIterator() ResourceIteratorI {
	return &ResourceIterator{MapIterator{data: d.resources, keys: reflect.ValueOf(d.resources).MapKeys(), currentIdx: 0, err: nil}}
}

func (d *Device) GetResource(id string) (ResourceI, error) {
	d.resourcesMutex.Lock()
	defer d.resourcesMutex.Unlock()
	if v, ok := d.resources[id].(ResourceI); ok {
		return v, nil
	}
	return nil, ErrNotExist
}

func (d *Device) GetDeviceId() (*uuid.UUID, error) {
	v, err := d.GetResource(DEVICE_URI)
	if err != nil {
		return nil, err
	}
	if rd, ok := v.(ResourceDeviceI); ok {
		return rd.GetDeviceId()
	}
	return nil, ErrNotExist
}

func NewDevice(rdevice ResourceDeviceI, rdiscovery ResourceDiscoveryI) (DeviceI, error) {
	if rdevice == nil || rdiscovery == nil || rdevice.GetId() == rdiscovery.GetId() {
		return nil, ErrInvalidParams
	}
	rs := make(map[interface{}]interface{}, 0)
	rs[rdevice.GetId()] = rdevice
	rs[rdiscovery.GetId()] = rdiscovery
	return &Device{resources: rs}, nil
}
