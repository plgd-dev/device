package ocfsdk

import (
	"sync"

	uuid "github.com/nu7hatch/gouuid"
)

type resourceIterator struct {
	MapIteratorMiddleware
}

func (i *resourceIterator) Value() ResourceI {
	v := i.ValueInterface()
	if v != nil {
		return v.(ResourceI)
	}
	return nil
}

type device struct {
	resources      map[interface{}]interface{}
	resourcesMutex sync.Mutex
}

func (d *device) AddResource(r ResourceI) error {
	if r == nil {
		return ErrInvalidParams
	}
	d.resourcesMutex.Lock()
	defer d.resourcesMutex.Unlock()
	if d.resources[r.GetID()] != nil {
		return ErrExist
	}
	newResources := make(map[interface{}]interface{})
	for key, val := range d.resources {
		newResources[key] = val
	}
	newResources[r.GetID()] = r
	d.resources = newResources
	return nil
}

func (d *device) DeleteResource(r ResourceI) error {
	if r == nil {
		return ErrInvalidParams
	}
	d.resourcesMutex.Lock()
	defer d.resourcesMutex.Unlock()
	if d.resources[r.GetID()] == nil {
		return ErrNotExist
	}
	newResources := make(map[interface{}]interface{})
	for key, val := range d.resources {
		if key != r.GetID() {
			newResources[key] = val
		}
	}
	d.resources = newResources
	return nil
}

func (d *device) NewResourceIterator() ResourceIteratorI {
	return &resourceIterator{MapIteratorMiddleware: MapIteratorMiddleware{i: NewMapIterator(d.resources)}}
}

func (d *device) GetResource(id string) (ResourceI, error) {
	d.resourcesMutex.Lock()
	defer d.resourcesMutex.Unlock()
	if v, ok := d.resources[id].(ResourceI); ok {
		return v, nil
	}
	return nil, ErrNotExist
}

func (d *device) GetDeviceID() (*uuid.UUID, error) {
	v, err := d.GetResource(deviceURI)
	if err != nil {
		return nil, err
	}
	if rd, ok := v.(ResourceDeviceI); ok {
		return rd.GetDeviceID()
	}
	return nil, ErrNotExist
}

//NewDevice creates device with resources device and discovery
func NewDevice(rdevice ResourceDeviceI, rdiscovery ResourceDiscoveryI) (DeviceI, error) {
	if rdevice == nil || rdiscovery == nil || rdevice.GetID() == rdiscovery.GetID() {
		return nil, ErrInvalidParams
	}
	rs := make(map[interface{}]interface{}, 0)
	rs[rdevice.GetID()] = rdevice
	rs[rdiscovery.GetID()] = rdiscovery
	return &device{resources: rs}, nil
}
