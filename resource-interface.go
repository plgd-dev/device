package ocfsdk

type ResourceCreateInterfaceI interface {
	Create(req RequestI, newResource ResourceI) (PayloadI, error)
}

type ResourceRetrieveInterfaceI interface {
	Retrieve(req RequestI, trans TransactionI) (PayloadI, error)
}

type ResourceUpdateInterfaceI interface {
	Update(req RequestI, errors []error) (PayloadI, error)
}

type ResourceDeleteInterfaceI interface {
	Delete(req RequestI, deletedResource ResourceI) (PayloadI, error)
}

type ResourceInterfaceI interface {
	IdI
}

type ResourceInterface struct {
	Id
}

type ResourceInterfaceBaseline struct {
	ResourceInterface
}

func (ri *ResourceInterfaceBaseline) Retrieve(req RequestI, transaction TransactionI) (PayloadI, error) {
	res := make(map[string]interface{})
	iface := make([]string, 0)
	for it := req.GetResource().NewResourceInterfaceIterator(); it.Value() != nil; it.Next() {
		if it.Value().GetId() != "" {
			iface = append(iface, it.Value().GetId())
		}
	}
	res["if"] = iface

	rt := make([]string, 0)
	for it := req.GetResource().NewResourceTypeIterator(); it.Value() != nil; it.Next() {
		rt = append(rt, it.Value().GetId())
	}
	res["rt"] = rt
	errors := make(map[string]error)
	for it := req.GetResource().NewResourceTypeIterator(); it.Value() != nil; it.Next() {
		for itA := it.Value().NewAttributeIterator(); itA.Value() != nil; itA.Next() {
			if value, err := itA.Value().GetValue(transaction); err == nil {
				res[itA.Value().GetId()] = value
			} else {
				errors[itA.Value().GetId()] = err
			}
		}
	}
	return res, nil
}

func (ri *ResourceInterfaceBaseline) Update(req RequestI, errors []error) (PayloadI, error) {
	return nil, nil
}
