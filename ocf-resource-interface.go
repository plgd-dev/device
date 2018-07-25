package main

import coap "github.com/ondrejtomcik/go-coap"

type OCFResourceCreateInterfaceI interface {
	Create(req OCFRequestI, newResource OCFResourceI) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceRetrieveInterfaceI interface {
	Retrieve(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceUpdateInterfaceI interface {
	Update(req OCFRequestI, changedAttributes []OCFAttributeI, errors []error) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceDeleteInterfaceI interface {
	Delete(req OCFRequestI, deletedResource OCFResourceI) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceInterfaceI interface {
	IdI
}

type OCFResourceInterface struct {
}

type OCFResourceInterfaceBaseline struct {
}

func (ri *OCFResourceInterfaceBaseline) GetId() string {
	return "oic.if.baseline"
}

func (ri *OCFResourceInterfaceBaseline) Retrieve(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error) {
	res := make(map[string]interface{})
	res["if"] = req.GetResource().GetResourceInterfaces()

	rt := make([]string, len(req.GetResource().GetResourceTypes()))
	for _, r := range req.GetResource().GetResourceTypes() {
		rt = append(rt, r.GetId())
	}
	res["ri"] = rt
	errors := make(map[string]error)
	for _, resourceType := range req.GetResource().GetResourceTypes() {
		for _, attribute := range resourceType.GetAttributes() {
			if value, err := attribute.GetValue(); err == nil {
				res[attribute.GetId()] = value
			} else {
				errors[attribute.GetId()] = err
			}
		}
	}
	return res, coap.Content, nil
}

func (ri *OCFResourceInterfaceBaseline) Update(req OCFRequestI, changedAttributes []OCFAttributeI, errors []error) (OCFPayloadI, coap.COAPCode, error) {
	return nil, coap.Changed, nil
}
