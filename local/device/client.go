package device

import (
	"context"
	"fmt"

	gocoap "github.com/go-ocf/go-coap"
	"github.com/go-ocf/kit/codec/coap"
	"github.com/go-ocf/sdk/local/resource"
	"github.com/go-ocf/sdk/schema"
)

func NewClient(client resourceClient, links schema.DeviceLinks) *Client {
	return &Client{client: client, links: links}
}

// Client uses an open connection to a device in order to query its details.
type Client struct {
	links  schema.DeviceLinks
	client resourceClient
}

type resourceClient interface {
	Get(ctx context.Context, deviceID, href string, codec resource.Codec, value interface{}, options ...func(gocoap.Message)) error
}

// QueryDevice queries device details for a device resource type.
func (c *Client) QueryDevice(ctx context.Context, resourceTypes ...string) (*schema.Device, error) {
	id := c.links.ID
	var d, nd schema.Device
	it := c.QueryResource(resourceTypes...)
	ok := it.Next(ctx, coap.CBORCodec{}, &d)
	if !ok {
		return nil, fmt.Errorf("could not get device details for %s: %v", id, it.Err)
	}
	if it.Next(ctx, coap.CBORCodec{}, &nd) {
		return nil, fmt.Errorf("too many resource links for %s %+v", id, resourceTypes)
	}
	return &d, nil
}

// QueryResource resolves URIs and returns an iterator for querying resources of a given type.
func (c *Client) QueryResource(resourceTypes ...string) *QueryResourceIterator {
	return &QueryResourceIterator{
		id:     c.links.ID,
		hrefs:  c.links.GetResourceHrefs(resourceTypes...),
		client: c.client,
	}
}

// QueryResourceIterator queries resources.
type QueryResourceIterator struct {
	Err    error
	id     string
	hrefs  []string
	i      int
	client resourceClient
}

// Next queries the next resource.
// Returns false when failed or having no more items.
// Check it.Err for errors.
func (it *QueryResourceIterator) Next(ctx context.Context, codec resource.Codec, v interface{}) bool {
	if it.i >= len(it.hrefs) {
		return false
	}

	err := it.client.Get(ctx, it.id, it.hrefs[it.i], codec, v)
	if err != nil {
		it.Err = fmt.Errorf("could not query the device %s: %v", it.id, err)
		return false
	}

	it.i++
	return true
}

// DeviceID returns id of device
func (c *Client) DeviceID() string {
	return c.links.ID
}

// GetResourceLinks returns all resource links.
func (c *Client) GetResourceLinks() []schema.ResourceLink {
	return c.links.Links
}

// GetEndpoints returns endpoints for a resource type.
// The endpoints are returned in order of priority.
func (c *Client) GetEndpoints(resourceType string) []schema.Endpoint {
	return c.links.GetEndpoints(resourceType)
}

// GetDeviceLinks returns device links.
func (c *Client) GetDeviceLinks() schema.DeviceLinks {
	return c.links
}
