package ocfsdk

type OCFPayloadI interface{}

type OCFRequestI interface {
	GetResource() OCFResourceI
	GetPayload() OCFPayloadI
	GetInterfaceId() string
	GetQueryParameters() []string
	GetPeerSession() interface{}
}

type OCFResourceTypeIteratorI interface {
	Next() bool
	Value() OCFResourceTypeI
	Error() error
}

type OCFResourceInterfaceIteratorI interface {
	Next() bool
	Value() OCFResourceInterfaceI
	Error() error
}

type OCFResourceI interface {
	OCFIdI

	IsDiscoverable() bool
	IsObserveable() bool
	NewResourceTypeIterator() OCFResourceTypeIteratorI
	NewResourceInterfaceIterator() OCFResourceInterfaceIteratorI
	NotifyObservers()
	OpenTransaction() (OCFTransactionI, error)
}
