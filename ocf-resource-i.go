package ocfsdk

type OCFPayloadI interface{}

type OCFRequestI interface {
	GetResource() OCFResourceI
	GetPayload() OCFPayloadI
	GetInterfaceId() string
	GetQueryParameters() []string
	GetPeerSession() interface{}
}

type OCFResourceI interface {
	OCFIdI

	IsDiscoverable() bool
	IsObserveable() bool
	GetResourceTypes() []OCFResourceTypeI
	GetResourceInterfaces() []OCFResourceInterfaceI
	NotifyObservers()
	OpenTransaction() (OCFTransactionI, error)
}
