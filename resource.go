package ocfsdk

import coap "github.com/ondrejtomcik/go-coap"

type ResourceTypeIterator struct {
	currentIdx int
	rt         []ResourceTypeI
	err        error
}

func (i *ResourceTypeIterator) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(i.rt) {
		return true
	}
	return false
}

func (i *ResourceTypeIterator) Value() ResourceTypeI {
	if i.currentIdx < len(i.rt) {
		return i.rt[i.currentIdx]
	}
	i.err = ErrInvalidIterator
	return nil
}

func (i *ResourceTypeIterator) Error() error {
	return i.err
}

type ResourceInterfaceIterator struct {
	currentIdx int
	ri         []ResourceInterfaceI
	err        error
}

func (i *ResourceInterfaceIterator) Next() bool {
	i.currentIdx++
	if i.currentIdx < len(i.ri) {
		return true
	}
	return false
}

func (i *ResourceInterfaceIterator) Value() ResourceInterfaceI {
	if i.currentIdx < len(i.ri) {
		return i.ri[i.currentIdx]
	}
	i.err = ErrInvalidIterator
	return nil
}

func (i *ResourceInterfaceIterator) Error() error {
	return i.err
}

type Resource struct {
	Id
	discoverable       bool
	observeable        bool
	resourceTypes      []ResourceTypeI
	resourceInterfaces []ResourceInterfaceI
	openTransaction    func() (TransactionI, error)
}

func (r *Resource) IsDiscoverable() bool {
	return r.discoverable
}

func (r *Resource) IsObserveable() bool {
	return r.observeable
}

func (r *Resource) NewResourceTypeIterator() ResourceTypeIteratorI {
	return &ResourceTypeIterator{currentIdx: 0, rt: r.resourceTypes}
}

func (r *Resource) NewResourceInterfaceIterator() ResourceInterfaceIteratorI {
	return &ResourceInterfaceIterator{currentIdx: 0, ri: r.resourceInterfaces}
}

func (r *Resource) OpenTransaction() (TransactionI, error) {
	if r.openTransaction != nil {
		return r.openTransaction()
	}
	return nil, ErrOperationNotSupported
}

func (r *Resource) NotifyObservers() {}

/*
func (r *Resource) Create(req RequestI) (PayloadI, coap.COAPCode, error) {
	for _, resourceInterface := range r.resourceInterfaces {
		if resourceInterface.GetId() == req.GetInterfaceId() {
			if ri, ok := resourceInterface.(ResourceCreateInterfaceI); ok {
				//create resource

				return ri.Create(req, newResource)
			}
		}
	}
	return nil, coap.NotImplemented, ErrInvalidInterface
}
*/

func (r *Resource) Retrieve(req RequestI) (PayloadI, coap.COAPCode, error) {
	if req == nil {
		return nil, coap.NotImplemented, ErrInvalidParams
	}
	for _, resourceInterface := range r.resourceInterfaces {
		if resourceInterface.GetId() == req.GetInterfaceId() {
			if ri, ok := resourceInterface.(ResourceRetrieveInterfaceI); ok {
				return ri.Retrieve(req)
			}
		}
	}
	return nil, coap.NotImplemented, ErrInvalidInterface
}

func (r *Resource) Update(req RequestI) (PayloadI, coap.COAPCode, error) {
	for _, resourceInterface := range r.resourceInterfaces {
		if resourceInterface.GetId() == req.GetInterfaceId() {
			if transaction, err := r.OpenTransaction(); err == nil {
				if ri, ok := resourceInterface.(ResourceUpdateInterfaceI); ok {
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
func (r *Resource) Delete(req RequestI) (PayloadI, coap.COAPCode, error) {
	for _, resourceInterface := range r.resourceInterfaces {
		if resourceInterface.GetId() == req.GetInterfaceId() {
			if ri, ok := resourceInterface.(ResourceDeleteI); ok {
				return ri.Delete(req)
			}
		}
	}
	return nil, coap.NotImplemented, ErrInvalidInterface
}
*/

func NewResource(id string, discoverable bool, observeable bool, resourceTypes []ResourceTypeI, resourceInterfaces []ResourceInterfaceI, openTransaction func() (TransactionI, error)) (ResourceI, error) {
	if len(id) == 0 || len(resourceTypes) == 0 {
		return nil, ErrInvalidParams
	}

	if resourceInterfaces == nil {
		resourceInterfaces = make([]ResourceInterfaceI, 0)
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
		resourceInterfaces = append(resourceInterfaces, &ResourceInterfaceBaseline{ResourceInterface: ResourceInterface{Id: Id{id: ""}}})
	}
	if !haveBaselineIf {
		resourceInterfaces = append(resourceInterfaces, &ResourceInterfaceBaseline{ResourceInterface: ResourceInterface{Id: Id{id: "oic.if.baseline"}}})
	}

	//without transaction
	if openTransaction == nil {
		openTransaction = func() (TransactionI, error) { return &DummyTransaction{}, nil }
	}

	return &Resource{Id: Id{id: id}, discoverable: discoverable, observeable: observeable, resourceTypes: resourceTypes, resourceInterfaces: resourceInterfaces, openTransaction: openTransaction}, nil
}
