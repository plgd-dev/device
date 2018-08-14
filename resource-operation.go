package ocfsdk

//ResourceOperationCreateFunc handler for create resource
type ResourceOperationCreateFunc func(RequestI) (ResourceI, error)

type resourceOperationCreate struct {
	create ResourceOperationCreateFunc
}

func (rc *resourceOperationCreate) Create(req RequestI) (PayloadI, error) {
	if rc.create == nil {
		return nil, ErrOperationNotSupported
	}
	for it := req.GetResource().NewResourceInterfaceIterator(); it.Value() != nil; it.Next() {
		if it.Value().GetID() == req.GetInterfaceID() {
			if ri, ok := it.Value().(ResourceCreateInterfaceI); ok {
				newResource, err := rc.create(req)
				if err != nil {
					return nil, err
				}
				return ri.Create(req, newResource)
			}
		}
	}
	return nil, ErrInvalidInterface
}

//OpenTransactionFunc handler for open transaction over resource
type OpenTransactionFunc func() (TransactionI, error)

type resourceOperationRetrieve struct {
	openTransaction OpenTransactionFunc
}

func (r *resourceOperationRetrieve) Retrieve(req RequestI) (PayloadI, error) {
	if ri, err := req.GetResource().GetResourceInterface(req.GetInterfaceID()); err == nil {
		if rir, ok := ri.(ResourceRetrieveInterfaceI); ok {
			var transaction TransactionI
			if r.openTransaction != nil {
				if transaction, err = r.openTransaction(); err != nil {
					return nil, err
				}
			}
			defer func() {
				if transaction != nil {
					transaction.Close()
				}
			}()
			return rir.Retrieve(req, transaction)
		}
	}
	return nil, ErrInvalidInterface
}

type resourceOperationUpdate struct {
	openTransaction OpenTransactionFunc
}

func (r *resourceOperationUpdate) Update(req RequestI) (PayloadI, error) {
	if ri, err := req.GetResource().GetResourceInterface(req.GetInterfaceID()); err == nil {
		if riu, ok := ri.(ResourceUpdateInterfaceI); ok {
			if transaction, err := r.openTransaction(); err == nil {
				defer func() {
					transaction.Close()
				}()
				reqMap := req.GetPayload().(map[string]interface{})
				errors := make([]error, 0)
				for key, value := range reqMap {
					for rit := req.GetResource().NewResourceTypeIterator(); rit.Value() != nil; rit.Next() {
						if attr, err := rit.Value().GetAttribute(key); err == nil {
							if err := attr.SetValue(transaction, value); err != nil {
								errors = append(errors, err)
							}
						}
					}
				}
				if err := transaction.Commit(); err != nil {
					errors = append(errors, err)
				}
				return riu.Update(req, transaction)
			}
		}
	}

	return nil, ErrInvalidInterface
}

//ResourceOperationDeleteFunc handler for delete resource
type ResourceOperationDeleteFunc func(RequestI) (deletedResource ResourceI, err error)

type resourceOperationDelete struct {
	delete ResourceOperationDeleteFunc
}

func (r *resourceOperationDelete) Delete(req RequestI) (PayloadI, error) {
	if r.delete == nil {
		return nil, ErrOperationNotSupported
	}
	if ri, err := req.GetResource().GetResourceInterface(req.GetInterfaceID()); err == nil {
		if rid, ok := ri.(ResourceDeleteInterfaceI); ok {
			dr, err := r.delete(req)
			if err != nil {
				return nil, err
			}
			return rid.Delete(req, dr)
		}
	}
	return nil, ErrInvalidInterface
}

//NewResourceOperationCreateDelete creates a resource operation that supports create, delete actions
func NewResourceOperationCreateDelete(create ResourceOperationCreateFunc, delete ResourceOperationDeleteFunc) ResourceOperationI {
	type ResourceOperationCreateDelete struct {
		resourceOperationCreate
		resourceOperationDelete
	}

	return &ResourceOperationCreateDelete{
		resourceOperationCreate: resourceOperationCreate{create},
		resourceOperationDelete: resourceOperationDelete{delete},
	}
}

//NewResourceOperationRetrieve creates a resource operation that support retrieve action
func NewResourceOperationRetrieve(openTransaction OpenTransactionFunc) ResourceOperationI {
	return &resourceOperationRetrieve{openTransaction}
}

//NewResourceOperationUpdate creates a resource operation that support update action
func NewResourceOperationUpdate(openTransaction OpenTransactionFunc) ResourceOperationI {
	return &resourceOperationUpdate{openTransaction}
}

//NewResourceOperationRetrieveUpdate creates a resource operation that support retrieve, update actions
func NewResourceOperationRetrieveUpdate(openTransaction OpenTransactionFunc) ResourceOperationI {
	type ResourceOperationRetrieveUpdate struct {
		resourceOperationRetrieve
		resourceOperationUpdate
	}
	return &ResourceOperationRetrieveUpdate{
		resourceOperationRetrieve: resourceOperationRetrieve{openTransaction},
		resourceOperationUpdate:   resourceOperationUpdate{openTransaction},
	}
}

//NewResourceOperationCRUD creates a resource operation that support create, retrieve, update, delete actions
func NewResourceOperationCRUD(create ResourceOperationCreateFunc, openTransaction OpenTransactionFunc, delete ResourceOperationDeleteFunc) ResourceOperationI {
	type ResourceOperationCRUD struct {
		resourceOperationCreate
		resourceOperationRetrieve
		resourceOperationUpdate
		resourceOperationDelete
	}
	return &ResourceOperationCRUD{
		resourceOperationCreate:   resourceOperationCreate{create},
		resourceOperationRetrieve: resourceOperationRetrieve{openTransaction},
		resourceOperationUpdate:   resourceOperationUpdate{openTransaction},
		resourceOperationDelete:   resourceOperationDelete{delete},
	}
}
