package ocfsdk

import (
	"sync"

	uuid "github.com/nu7hatch/gouuid"
)

type ResourceIterator struct {
	MapIteratorMiddleware
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
	newResources := make(map[interface{}]interface{})
	for key, val := range d.resources {
		newResources[key] = val
	}
	newResources[r.GetId()] = r
	d.resources = newResources
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
	newResources := make(map[interface{}]interface{})
	for key, val := range d.resources {
		if key != r.GetId() {
			newResources[key] = val
		}
	}
	d.resources = newResources
	return nil
}

func (d *Device) NewResourceIterator() ResourceIteratorI {
	return &ResourceIterator{MapIteratorMiddleware: MapIteratorMiddleware{i: NewMapIterator(d.resources)}}
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
