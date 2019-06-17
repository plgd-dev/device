package local

import (
	"context"
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/ocf"
	"github.com/go-ocf/kit/net"
	kitNetCoap "github.com/go-ocf/kit/net/coap"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"
)

type coapClient struct {
	*kitNetCoap.Client
	scheme string
}

func NewCoapClient(clientConn *gocoap.ClientConn, scheme string) *coapClient {
	return &coapClient{Client: kitNetCoap.NewClient(clientConn), scheme: scheme}
}

// IsIotivity detects if server is iotivity 2.0.1-RC0.
// Tested against iotivty 2.0.1-RC0 and iotivity-lite with revision 04471e61bbf8b936e4531f07f5b4a6fc2b0bc966.
func IsIotivity(ctx context.Context, c *kitNetCoap.Client) (bool, error) {
	href := "/oic/res"
	errMsg := "cannot determine whether it is iotivity:"
	req, err := c.NewGetRequest("/oic/res")
	if err != nil {
		return false, fmt.Errorf("could create get request %s: %v", href, err)
	}
	req.AddOption(gocoap.Accept, gocoap.AppOcfCbor)
	resp, err := c.ExchangeWithContext(ctx, req)
	if err != nil {
		return false, fmt.Errorf("could not query %s: %v", href, err)
	}
	if resp.Code() != gocoap.Content {
		return false, fmt.Errorf(errMsg+" request failed: %s", ocf.Dump(resp))
	}

	cf := resp.Option(gocoap.ContentFormat)
	if cf == nil {
		return false, fmt.Errorf(errMsg + " content format not found")
	}
	mt, _ := cf.(gocoap.MediaType)
	switch mt {
	case gocoap.AppCBOR:
		return true, nil
	case gocoap.AppOcfCbor:
		return false, nil
	}

	return false, fmt.Errorf(errMsg+" unknown content format %v", mt)
}

func (c *coapClient) GetDeviceLinks(ctx context.Context, deviceID string) (device schema.DeviceLinks, _ error) {
	var devices []schema.DeviceLinks
	err := c.GetResourceWithCodec(ctx, "/oic/res", resource.DiscoveryResourceCodec{}, &devices)
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
		addr, err := net.Parse(c.scheme, c.Client.RemoteAddr())
		if err != nil {
			return device, fmt.Errorf("invalid address of device %s: %v", device.ID, err)
		}

		links = append(links, link.PatchEndpoint(addr))
	}
	//filter device links with endpoints
	device.Links = resource.FilterResourceLinksWithEndpoints(links)

	return device, nil
}
