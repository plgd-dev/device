package ocfsdk

type PayloadI interface{}

type RequestI interface {
	GetResource() ResourceI
	GetPayload() PayloadI
	GetInterfaceId() string
	GetQueryParameters() []string
	GetPeerSession() interface{}
}

type ResourceTypeIteratorI interface {
	Next() bool
	Value() ResourceTypeI
	Error() error
}

type ResourceInterfaceIteratorI interface {
	Next() bool
	Value() ResourceInterfaceI
	Error() error
}

type ResourceI interface {
	IdI

	IsDiscoverable() bool
	IsObserveable() bool
	NewResourceTypeIterator() ResourceTypeIteratorI
	NewResourceInterfaceIterator() ResourceInterfaceIteratorI
	NotifyObservers()
	OpenTransaction() (TransactionI, error)
}
