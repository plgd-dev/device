package service

import (
	"bytes"

	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/schema/device"
	"github.com/plgd-dev/device/v2/schema/interfaces"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
)

type DeviceResource struct {
	*Resource
	device *Device
}

func NewDeviceResource(dev *Device) *DeviceResource {
	d := &DeviceResource{
		device: dev,
	}
	d.Resource = NewResource(device.ResourceURI, d.Get, nil, dev.cfg.ResourceTypes, []string{interfaces.OC_IF_BASELINE, interfaces.OC_IF_R})
	return d
}

/*
func (d *DeviceResource) getEndpoint(request *Request) (string, net.IP) {
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

func (d *DeviceResource) Get(request *Request) (*pool.Message, error) {
	//ep, ip := d.getEndpoint(request)
	// /fmt.Printf("DiscoveryResource ep=%v index=%v ip=%v\n", ep, request.ControlMessage().IfIndex, ip)
	v := device.Device{
		ID:                    d.device.cfg.ID,
		Name:                  d.device.cfg.Name,
		ProtocolIndependentID: d.device.cfg.ProtocolIndependentID,
		DataModelVersion:      "ocf.res.1.3.0",
		SpecificationVersion:  "ocf.2.0.5",
	}
	if request.Interface() == interfaces.OC_IF_BASELINE {
		v.ResourceTypes = d.Resource.ResourceTypes
		v.Interfaces = d.Resource.ResourceInterfaces
	}

	res := pool.NewMessage(request.Context())
	res.SetCode(codes.Content)
	res.SetContentFormat(message.AppOcfCbor)
	/*
		res.SetControlMessage(&coapNet.ControlMessage{
			IfIndex: request.ControlMessage().IfIndex,
			Src:     ip,
		})
	*/
	data, err := cbor.Encode(v)
	if err != nil {
		return nil, err
	}
	res.SetBody(bytes.NewReader(data))
	return res, nil
}
