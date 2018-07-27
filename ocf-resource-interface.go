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
	for _, r := range req.GetResource().GetResourceInterfaces() {
		if r.GetId() != "" {
			iface = append(iface, r.GetId())
		}
	}
	res["if"] = iface

	rt := make([]string, 0)
	for _, r := range req.GetResource().GetResourceTypes() {
		rt = append(rt, r.GetId())
	}
	res["rt"] = rt
	errors := make(map[string]error)
	for _, resourceType := range req.GetResource().GetResourceTypes() {
		for _, attribute := range resourceType.GetAttributes() {
			if value, err := attribute.GetValue(transaction); err == nil {
				res[attribute.GetId()] = value
			} else {
				errors[attribute.GetId()] = err
			}
		}
	}
	transaction.Drop()
	return res, coap.Content, nil
}

func (ri *OCFResourceInterfaceBaseline) Update(req OCFRequestI, errors []error) (OCFPayloadI, coap.COAPCode, error) {
	return nil, coap.Changed, nil
}
