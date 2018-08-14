package ocfsdk

type resourceInterface struct {
	id
}

type resourceInterfaceBaseline struct {
	resourceInterface
}

func (ri *resourceInterfaceBaseline) Retrieve(req RequestI, transaction TransactionI) (PayloadI, error) {
	res := make(map[string]interface{})
	iface := make([]string, 0)
	for it := req.GetResource().NewResourceInterfaceIterator(); it.Value() != nil; it.Next() {
		if it.Value().GetID() != "" {
			iface = append(iface, it.Value().GetID())
		}
	}
	res["if"] = iface

	rt := make([]string, 0)
	for it := req.GetResource().NewResourceTypeIterator(); it.Value() != nil; it.Next() {
		rt = append(rt, it.Value().GetID())
	}
	res["rt"] = rt
	errors := make(map[string]error)
	for it := req.GetResource().NewResourceTypeIterator(); it.Value() != nil; it.Next() {
		for itA := it.Value().NewAttributeIterator(); itA.Value() != nil; itA.Next() {
			if value, err := itA.Value().GetValue(transaction); err == nil {
				res[itA.Value().GetID()] = value
			} else {
				errors[itA.Value().GetID()] = err
			}
		}
	}
	return res, nil
}

func (ri *resourceInterfaceBaseline) Update(req RequestI, transaction TransactionI) (PayloadI, error) {
	return nil, nil
}

//NewResourceInterfaceBaseline creates a resource interface with the name
func NewResourceInterfaceBaseline(name string) ResourceInterfaceI {
	return &resourceInterfaceBaseline{resourceInterface: resourceInterface{id: id{id: name}}}
}
