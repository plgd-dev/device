package main

import coap "github.com/ondrejtomcik/go-coap"

type OCFResourceCreateI interface {
	Create(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceRetrieveI interface {
	Retrieve(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceUpdateI interface {
	Update(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceDeleteI interface {
	Delete(req OCFRequestI) (OCFPayloadI, coap.COAPCode, error)
}

type OCFResourceCRUDI interface {
	OCFResourceCreateI
	OCFResourceRetrieveI
	OCFResourceUpdateI
	OCFResourceDeleteI
}
