package ocfsdk

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
	OpenTransaction() (OCFTransactionI, error)
}

type OCFResource struct {
	Id                 IdI
	discoverable       bool
	observeable        bool
	resourceTypes      []OCFResourceTypeI
	resourceInterfaces []OCFResourceInterfaceI
	openTransaction    func() (OCFTransactionI, error)
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

func (r *OCFResource) OpenTransaction() (OCFTransactionI, error) {
	if r.openTransaction != nil {
		return r.openTransaction()
	}
	return nil, ErrOperationNotSupported
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
			if transaction, err := r.OpenTransaction(); err != nil {
				if ri, ok := resourceInterface.(OCFResourceUpdateInterfaceI); ok {
					reqMap := req.GetPayload().(map[string]interface{})
					errors := make([]error, 10)
					for key, value := range reqMap {
						for _, resourceType := range req.GetResource().GetResourceTypes() {
							for _, attribute := range resourceType.GetAttributes() {
								if attribute.GetId() == key {
									if err := attribute.SetValue(transaction, value); err != nil {
										errors = append(errors, err)
									}
								}
							}
						}
					}
					if err := transaction.Commit(); err != nil {
						errors = append(errors, err)
					}
					return ri.Update(req, errors)
				}
				transaction.Drop()
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
