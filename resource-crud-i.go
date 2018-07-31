package ocfsdk

import coap "github.com/go-ocf/go-coap"

type ResourceCreateI interface {
	Create(req RequestI) (PayloadI, coap.COAPCode, error)
}

type ResourceRetrieveI interface {
	Retrieve(req RequestI) (PayloadI, coap.COAPCode, error)
}

type ResourceUpdateI interface {
	Update(req RequestI) (PayloadI, coap.COAPCode, error)
}

type ResourceDeleteI interface {
	Delete(req RequestI) (PayloadI, coap.COAPCode, error)
}

type ResourceCRUDI interface {
	ResourceCreateI
	ResourceRetrieveI
	ResourceUpdateI
	ResourceDeleteI
}
