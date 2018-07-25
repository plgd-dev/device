package main

import coap "github.com/ondrejtomcik/go-coap"

type IdI interface {
	GetId() string
}

type Id struct {
	Id string
}

func (i *Id) GetId() string {
	return i.Id
}

type OCFPayloadI interface{}

type OCFRequestI interface {
	GetResource() OCFResourceI
	GetPayload() OCFPayloadI
	GetInterfaceId() string
	GetQueryParameters() []string
	GetPeerSession() coap.Session
}

type OCFResourceI interface {
	IdI

	IsDiscoverable() bool
	IsObserveable() bool
	GetResourceTypes() []OCFResourceTypeI
	GetResourceInterfaces() []OCFResourceInterfaceI
	NotifyObservers()
}

type OCFResource struct {
	Id                 IdI
	discoverable       bool
	observeable        bool
	resourceTypes      []OCFResourceTypeI
	resourceInterfaces []OCFResourceInterfaceI
}

func (r *OCFResource) IsDiscoverable() bool {
	return r.discoverable
}

func (r *OCFResource) IsObserveable() bool {
	return r.observeable
}

func (r *OCFResource) GetId() string {
	return r.Id.GetId()
}

func (r *OCFResource) GetResourceTypes() []OCFResourceTypeI {
	return r.resourceTypes
}

func (r *OCFResource) GetResourceInterfaces() []OCFResourceInterfaceI {
	return r.resourceInterfaces
}

/*
func (r *OCFResource) Create(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error) {
	for _, resourceInterface := range r.resourceInterfaces {
		if resourceInterface.GetId() == req.GetInterfaceId() {
			if ri, ok := resourceInterface.(OCFResourceCreateInterfaceI); ok {
				//create resource

				return ri.Create(req, newResource)
			}
		}
	}
	return nil, coap.NotImplemented, ErrInvalidInterface
}
*/

func (r *OCFResource) Retrieve(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error) {
	for _, resourceInterface := range r.resourceInterfaces {
		if resourceInterface.GetId() == req.GetInterfaceId() {
			if ri, ok := resourceInterface.(OCFResourceRetrieveInterfaceI); ok {
				return ri.Retrieve(req)
			}
		}
	}
	return nil, coap.NotImplemented, ErrInvalidInterface
}

func (r *OCFResource) Update(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error) {
	for _, resourceInterface := range r.resourceInterfaces {
		if resourceInterface.GetId() == req.GetInterfaceId() {
			if ri, ok := resourceInterface.(OCFResourceUpdateInterfaceI); ok {
				reqMap := req.GetPayload().(map[string]interface{})
				errors := make([]error, 10)
				changedAttributes := make([]OCFAttributeI, 10)
				for _, value := range reqMap {
					for _, resourceType := range req.GetResource().GetResourceTypes() {
						for _, attribute := range resourceType.GetAttributes() {
							if changed, err := attribute.SetValue(value); err != nil {
								errors = append(errors, err)
							} else if changed {
								changedAttributes = append(changedAttributes, attribute)
							}
						}
					}
				}
				return ri.Update(req, changedAttributes, errors)
			}
		}
	}

	return nil, coap.NotImplemented, ErrInvalidInterface
}

/*
func (r *OCFResource) Delete(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error) {
	for _, resourceInterface := range r.resourceInterfaces {
		if resourceInterface.GetId() == req.GetInterfaceId() {
			if ri, ok := resourceInterface.(OCFResourceDeleteI); ok {
				return ri.Delete(req)
			}
		}
	}
	return nil, coap.NotImplemented, ErrInvalidInterface
}
*/
