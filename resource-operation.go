package ocfsdk

type ResourceOperationCreateFunc func(RequestI) (ResourceI, error)

type ResourceOperationCreate struct {
	create ResourceOperationCreateFunc
}

func (rc *ResourceOperationCreate) Create(req RequestI) (PayloadI, error) {
	if rc.create == nil {
		return nil, ErrOperationNotSupported
	}
	for it := req.GetResource().NewResourceInterfaceIterator(); it.Value() != nil; it.Next() {
		if it.Value().GetId() == req.GetInterfaceId() {
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

type OpenTransactionFunc func() (TransactionI, error)

type ResourceOperationRetrieve struct {
	openTransaction OpenTransactionFunc
}

func (r *ResourceOperationRetrieve) Retrieve(req RequestI) (PayloadI, error) {
	if ri, err := req.GetResource().GetResourceInterface(req.GetInterfaceId()); err == nil {
		if rir, ok := ri.(ResourceRetrieveInterfaceI); ok {
			var t TransactionI
			if r.openTransaction != nil {
				if t, err = r.openTransaction(); err != nil {
					return nil, err
				}
			}
			defer func() {
				if t != nil {
					t.Drop()
				}
			}()
			return rir.Retrieve(req, t)
		}
	}
	return nil, ErrInvalidInterface
}

type ResourceOperationUpdate struct {
	openTransaction OpenTransactionFunc
}

func (r *ResourceOperationUpdate) Update(req RequestI) (PayloadI, error) {
	if ri, err := req.GetResource().GetResourceInterface(req.GetInterfaceId()); err == nil {
		if riu, ok := ri.(ResourceUpdateInterfaceI); ok {
			if transaction, err := r.openTransaction(); err == nil {
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
				return riu.Update(req, errors)
			}
		}
	}

	return nil, ErrInvalidInterface
}

type ResourceOperationDeleteFunc func(RequestI) (deletedResource ResourceI, err error)

type ResourceOperationDelete struct {
	delete ResourceOperationDeleteFunc
}

func (r *ResourceOperationDelete) Delete(req RequestI) (PayloadI, error) {
	if r.delete == nil {
		return nil, ErrOperationNotSupported
	}
	if ri, err := req.GetResource().GetResourceInterface(req.GetInterfaceId()); err == nil {
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

func NewResourceOperationCreateDelete(create ResourceOperationCreateFunc, delete ResourceOperationDeleteFunc) ResourceOperationI {
	type ResourceOperationCreateDelete struct {
		ResourceOperationCreate
		ResourceOperationDelete
	}

	return &ResourceOperationCreateDelete{
		ResourceOperationCreate: ResourceOperationCreate{create},
		ResourceOperationDelete: ResourceOperationDelete{delete},
	}
}

func NewResourceOperationRetrieve(openTransaction OpenTransactionFunc) ResourceOperationI {
	return &ResourceOperationRetrieve{openTransaction}
}

func NewResourceOperationUpdate(openTransaction OpenTransactionFunc) ResourceOperationI {
	return &ResourceOperationUpdate{openTransaction}
}

func NewResourceOperationRetrieveUpdate(openTransaction OpenTransactionFunc) ResourceOperationI {
	type ResourceOperationRetrieveUpdate struct {
		ResourceOperationRetrieve
		ResourceOperationUpdate
	}
	return &ResourceOperationRetrieveUpdate{
		ResourceOperationRetrieve: ResourceOperationRetrieve{openTransaction},
		ResourceOperationUpdate:   ResourceOperationUpdate{openTransaction},
	}
}

func NewResourceOperationCRUD(create ResourceOperationCreateFunc, openTransaction OpenTransactionFunc, delete ResourceOperationDeleteFunc) ResourceOperationI {
	type ResourceOperationCRUD struct {
		ResourceOperationCreate
		ResourceOperationRetrieve
		ResourceOperationUpdate
		ResourceOperationDelete
	}
	return &ResourceOperationCRUD{
		ResourceOperationCreate:   ResourceOperationCreate{create},
		ResourceOperationRetrieve: ResourceOperationRetrieve{openTransaction},
		ResourceOperationUpdate:   ResourceOperationUpdate{openTransaction},
		ResourceOperationDelete:   ResourceOperationDelete{delete},
	}
}
