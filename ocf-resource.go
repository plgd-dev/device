package ocfsdk

import coap "github.com/ondrejtomcik/go-coap"

type OCFResourceTypeIterator struct {
	currentIdx int
	rt         []OCFResourceTypeI
	err        error
}

func (i *OCFResourceTypeIterator) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(i.rt) {
		return true
	}
	return false
}

func (i *OCFResourceTypeIterator) Value() OCFResourceTypeI {
	if i.currentIdx < len(i.rt) {
		return i.rt[i.currentIdx]
	}
	i.err = ErrInvalidIterator
	return nil
}

func (i *OCFResourceTypeIterator) Error() error {
	return i.err
}

type OCFResourceInterfaceIterator struct {
	currentIdx int
	ri         []OCFResourceInterfaceI
	err        error
}

func (i *OCFResourceInterfaceIterator) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(i.ri) {
		return true
	}
	return false
}

func (i *OCFResourceInterfaceIterator) Value() OCFResourceInterfaceI {
	if i.currentIdx < len(i.ri) {
		return i.ri[i.currentIdx]
	}
	i.err = ErrInvalidIterator
	return nil
}

func (i *OCFResourceInterfaceIterator) Error() error {
	return i.err
}

type OCFResource struct {
	OCFId
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

func (r *OCFResource) NewResourceTypeIterator() OCFResourceTypeIteratorI {
	return &OCFResourceTypeIterator{currentIdx: 0, rt: r.resourceTypes}
}

func (r *OCFResource) NewResourceInterfaceIterator() OCFResourceInterfaceIteratorI {
	return &OCFResourceInterfaceIterator{currentIdx: 0, ri: r.resourceInterfaces}
}

func (r *OCFResource) OpenTransaction() (OCFTransactionI, error) {
	if r.openTransaction != nil {
		return r.openTransaction()
	}
	return nil, ErrOperationNotSupported
}

func (r *OCFResource) NotifyObservers() {}

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
	if req == nil {
		return nil, coap.NotImplemented, ErrInvalidParams
	}
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
			if transaction, err := r.OpenTransaction(); err == nil {
				if ri, ok := resourceInterface.(OCFResourceUpdateInterfaceI); ok {
					reqMap := req.GetPayload().(map[string]interface{})
					errors := make([]error, 10)
					for key, value := range reqMap {
						for _, resourceType := range r.resourceTypes {
							for it := resourceType.NewAttributeIterator(); it.Value() != nil; it.Next() {
								attribute := it.Value()
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

func NewResource(id string, discoverable bool, observeable bool, resourceTypes []OCFResourceTypeI, resourceInterfaces []OCFResourceInterfaceI, openTransaction func() (OCFTransactionI, error)) (OCFResourceI, error) {
	if len(id) == 0 || len(resourceTypes) == 0 {
		return nil, ErrInvalidParams
	}

	if resourceInterfaces == nil {
		resourceInterfaces = make([]OCFResourceInterfaceI, 0)
	}

	haveDefaultIf := false
	haveBaselineIf := false

	for _, i := range resourceInterfaces {
		if i.GetId() == "" {
			haveDefaultIf = true
		}
		if i.GetId() == "oic.if.baseline" {
			haveBaselineIf = true
		}
		if haveDefaultIf && haveBaselineIf {
			break
		}
	}
	if !haveDefaultIf {
		resourceInterfaces = append(resourceInterfaces, &OCFResourceInterfaceBaseline{OCFResourceInterface: OCFResourceInterface{OCFId: OCFId{id: ""}}})
	}
	if !haveBaselineIf {
		resourceInterfaces = append(resourceInterfaces, &OCFResourceInterfaceBaseline{OCFResourceInterface: OCFResourceInterface{OCFId: OCFId{id: "oic.if.baseline"}}})
	}

	//without transaction
	if openTransaction == nil {
		openTransaction = func() (OCFTransactionI, error) { return &OCFDummyTransaction{}, nil }
	}

	return &OCFResource{OCFId: OCFId{id: id}, discoverable: discoverable, observeable: observeable, resourceTypes: resourceTypes, resourceInterfaces: resourceInterfaces, openTransaction: openTransaction}, nil
}
