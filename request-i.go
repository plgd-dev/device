package ocfsdk

//RequestI defines interface of request
type RequestI interface {
	//GetResource returns current resource where request is processed
	GetResource() ResourceI
	//GetPayload returns payload of current request
	GetPayload() PayloadI
	//GetInterfaceID returns name of resource interface that must be used for creating response
	GetInterfaceID() string
	//GetInterfaceID returns array of query parameters
	GetQueryParameters() []string
	//GetDevice returns device where request is processed
	GetDevice() DeviceI
}
