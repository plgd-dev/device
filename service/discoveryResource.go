package service

import (
	"bytes"

	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/device/v2/schema/resources"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

type DiscoveryResource struct {
	device *Device
	*Resource
}

func NewDiscoveryResource(device *Device) *DiscoveryResource {
	d := &DiscoveryResource{
		device: device,
	}
	d.Resource = NewResource(resources.ResourceURI, d.Get, nil, []string{resources.ResourceType}, []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_R})
	return d
}

/*
	func (d *DiscoveryResource) getEndpoint(request *Request) (string, net.IP) {
		if request.ControlMessage().Dst.IsMulticast() {
			iface, err := net.InterfaceByIndex(request.ControlMessage().GetIfIndex())
			if err == nil {
				addrs, err := iface.Addrs()
				if err == nil {
					for _, addr := range addrs {
						if ipnet, ok := addr.(*net.IPNet); ok {
							if ipnet.IP.To4() != nil {
								return fmt.Sprintf("coap://%v:%v", ipnet.IP.String(), d.device.listener.LocalAddr().(*net.UDPAddr).Port), ipnet.IP
							}
						}
					}
				}
			}
		}
		return fmt.Sprintf("coap://%v:%v", request.ControlMessage().Dst.String(), d.device.listener.LocalAddr().(*net.UDPAddr).Port), request.ControlMessage().Dst
	}
*/
func (d *DiscoveryResource) Get(request *Request) (*pool.Message, error) {
	//ep, ip := d.getEndpoint(request)
	//fmt.Printf("DiscoveryResource ep=%v index=%v ip=%v\n", ep, request.ControlMessage().IfIndex, ip)
	resourceTypes := request.ResourceTypes()
	resources := make(schema.ResourceLinks, 0, d.device.resources.Length())
	d.device.resources.Range(func(key string, resource *Resource) bool {
		if len(resourceTypes) > 0 && !resource.HasResourceTypes(resourceTypes) {
			return true
		}
		resources = append(resources, schema.ResourceLink{
			Href:          key,
			ResourceTypes: resource.ResourceTypes,
			Interfaces:    resource.ResourceInterfaces,
			Policy: &schema.Policy{
				BitMask: schema.Discoverable,
			},
			Anchor:    "ocf://" + d.device.cfg.ID + key,
			DeviceID:  d.device.cfg.ID,
			Endpoints: d.device.getEndpoints(),
		})
		return true
	})
	res := pool.NewMessage(request.Context())
	res.SetCode(codes.Content)
	res.SetContentFormat(message.AppOcfCbor)
	/*
		res.SetControlMessage(&coapNet.ControlMessage{
			IfIndex: request.ControlMessage().IfIndex,
			Src:     ip,
		})
	*/
	data, err := cbor.Encode(resources)
	if err != nil {
		return nil, err
	}
	res.SetBody(bytes.NewReader(data))
	return res, nil
}
