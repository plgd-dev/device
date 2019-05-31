package local

import (
	"context"
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/kit/net"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"
)

type CoapClient struct {
	clientConn *gocoap.ClientConn
}

func NewCoapClient(clientConn *gocoap.ClientConn) *CoapClient {
	return &CoapClient{clientConn: clientConn}
}

func (c *CoapClient) UpdateResourceCBOR(
	ctx context.Context,
	href string,
	interfaceFilter string,
	request interface{},
	response interface{},
) error {
	return c.UpdateResource(ctx, href, interfaceFilter, coap.CBORCodec{}, request, response)
}

func (c *CoapClient) UpdateResource(
	ctx context.Context,
	href string,
	interfaceFilter string,
	codec resource.Codec,
	request interface{},
	response interface{},
) error {
	var options []func(gocoap.Message)
	if interfaceFilter != "" {
		options = append(options, func(req gocoap.Message) {
			req.AddOption(gocoap.URIQuery, "if="+interfaceFilter)
		})
	}

	return resource.COAPPost(ctx, c.clientConn, href, codec, request, response, options...)
}

func (c *CoapClient) GetResourceCBOR(
	ctx context.Context,
	href string,
	interfaceFilter string,
	response interface{},
) error {
	return c.GetResource(ctx, href, interfaceFilter, coap.CBORCodec{}, response)
}

func (c *CoapClient) GetResource(
	ctx context.Context,
	href string,
	interfaceFilter string,
	codec resource.Codec,
	response interface{},
) error {
	var options []func(gocoap.Message)
	if interfaceFilter != "" {
		options = append(options, func(req gocoap.Message) {
			req.AddOption(gocoap.URIQuery, "if="+interfaceFilter)
		})
	}

	return resource.COAPGet(ctx, c.clientConn, href, codec, response, options...)
}

func (c *CoapClient) GetDeviceLinks(ctx context.Context, deviceID string) (device schema.DeviceLinks, _ error) {
	var devices []schema.DeviceLinks
	err := c.GetResourceCBOR(ctx, "/oic/res", "", &devices)
	if err != nil {
		return device, err
	}
	for _, d := range devices {
		if d.ID == deviceID {
			device = d
		}
	}
	if device.ID != deviceID {
		return device, fmt.Errorf("cannot get device links: not found")
	}

	links := make([]schema.ResourceLink, 0, len(device.Links))
	for _, link := range device.Links {
		addr, err := net.Parse(c.clientConn.RemoteAddr())
		if err != nil {
			return device, fmt.Errorf("invalid address of device %s: %v", device.ID, err)
		}
		links = append(links, link.PatchEndpoint(addr))
	}
	device.Links = links

	return device, nil
}
