package ocfsdk

type RequestI interface {
	GetResource() ResourceI
	GetPayload() PayloadI
	GetInterfaceId() string
	GetQueryParameters() []string
	GetPeerSession() interface{}
	GetDevice() DeviceI
}
