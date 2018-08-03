package ocfsdk

import coap "github.com/go-ocf/go-coap"

type ResourceOperationI interface {
}

type ResourceOperationCreateI interface {
	Create(req RequestI) (PayloadI, coap.COAPCode, error)
}

type ResourceOperationRetrieveI interface {
	Retrieve(req RequestI) (PayloadI, coap.COAPCode, error)
}

type ResourceOperationUpdateI interface {
	Update(req RequestI) (PayloadI, coap.COAPCode, error)
}

type ResourceOperationDeleteI interface {
	Delete(req RequestI) (PayloadI, coap.COAPCode, error)
}
