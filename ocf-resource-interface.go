package ocfsdk

import coap "github.com/ondrejtomcik/go-coap"

type OCFResourceCreateInterfaceI interface {
	Create(req OCFRequestI, newResource OCFResourceI) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceRetrieveInterfaceI interface {
	Retrieve(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceUpdateInterfaceI interface {
	Update(req OCFRequestI, errors []error) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceDeleteInterfaceI interface {
	Delete(req OCFRequestI, deletedResource OCFResourceI) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceInterfaceI interface {
	OCFIdI
}

type OCFResourceInterface struct {
	OCFId
}

type OCFResourceInterfaceBaseline struct {
	OCFResourceInterface
}

func (ri *OCFResourceInterfaceBaseline) Retrieve(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error) {
	transaction, err := req.GetResource().OpenTransaction()
	if err != nil {
		return nil, coap.InternalServerError, err
	}

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
	transaction.Drop()
	return res, coap.Content, nil
}

func (ri *OCFResourceInterfaceBaseline) Update(req OCFRequestI, errors []error) (OCFPayloadI, coap.COAPCode, error) {
	return nil, coap.Changed, nil
}
