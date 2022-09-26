package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/message/status"
)

func getLinksRefDevice(ctx context.Context, refDev *RefDevice, disableUDPEndpoints bool) (schema.ResourceLinks, error) {
	endpoints := refDev.GetEndpoints()
	links, err := refDev.GetResourceLinks(ctx, endpoints)
	if err != nil {
		return nil, err
	}
	return patchResourceLinksEndpoints(links, disableUDPEndpoints), nil
}

func getRefDeviceFromCache(ctx context.Context, deviceCache *refDeviceCache,
	deviceID string, disableUDPEndpoints bool,
) (*RefDevice, schema.ResourceLinks, bool) {
	refDev, ok := deviceCache.GetDevice(ctx, deviceID)
	if ok {
		links, err := getLinksRefDevice(ctx, refDev, disableUDPEndpoints)

		if err != nil {
			// Don't remove devices found by IP, the device is probably offline
			// and we will be not able to reestablish the connection when it will
			// come back online
			refDev.Device().Close(ctx)
			if refDev.Device().FoundByIP() == "" {
				deviceCache.RemoveDevice(ctx, refDev.DeviceID(), refDev)
				refDev.Release(ctx)
			}
			return nil, nil, false
		}
		return refDev, links, true
	}
	return nil, nil, false
}

// GetRefDeviceByIP gets the device directly via IP address and multicast listen port 5683. After using it, call device.Release to free resources.
func (c *Client) GetRefDeviceByIP(
	ctx context.Context,
	ip string,
) (*RefDevice, schema.ResourceLinks, error) {
	fmt.Println("######################### GetRefDeviceByIP")
	dev, err := c.client.GetDeviceByIP(ctx, ip)
	if err != nil {
		for k, v := range c.deviceCache.GetContent() {
			if v == ip {
				e, ok := c.deviceCache.GetDevice(ctx, k)
				if ok {
					e.Device().Close(ctx)
					e.Release(ctx)
				}
				break
			}
		}
		fmt.Println(err)
		return nil, nil, err
	}

	newRefDev := NewRefDevice(dev)
	refDev, stored := c.deviceCache.TryStoreDeviceToTemporaryCache(newRefDev)

	if !stored {
		newRefDev.Release(ctx)
	}
	links, err := getLinksRefDevice(ctx, refDev, c.disableUDPEndpoints)
	if err != nil {
		fmt.Println(err)
		deviceID := refDev.DeviceID()
		// Don't remove devices found by IP, the device is probably offline
		// and we will be not able to reestablish the connection when it will
		// come back online
		refDev.Device().Close(ctx)
		if refDev.Device().FoundByIP() == "" {
			c.deviceCache.RemoveDevice(ctx, refDev.DeviceID(), refDev)
			refDev.Release(ctx)
		}
		return nil, nil, fmt.Errorf("cannot get links for device %v: %w", deviceID, err)
	}
	return refDev, patchResourceLinksEndpoints(links, c.disableUDPEndpoints), nil
}

// GetRefDevice returns device, after using call device.Release to free resources.
func (c *Client) GetRefDevice(
	ctx context.Context,
	deviceID string,
	opts ...GetDeviceOption,
) (*RefDevice, schema.ResourceLinks, error) {
	cfg := getDeviceOptions{
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnGetDevice(cfg)
	}
	refDev, links, ok := getRefDeviceFromCache(ctx, c.deviceCache, deviceID, c.disableUDPEndpoints)
	if ok {
		return refDev, links, nil
	}
	dev, err := c.client.GetDeviceByMulticast(ctx, deviceID, cfg.discoveryConfiguration)
	if err != nil {
		return nil, nil, err
	}

	newRefDev := NewRefDevice(dev)
	refDev, stored := c.deviceCache.TryStoreDeviceToTemporaryCache(newRefDev)
	if !stored {
		newRefDev.Release(ctx)
	}
	links, err = getLinksRefDevice(ctx, refDev, c.disableUDPEndpoints)
	if err != nil {
		// Don't remove devices found by IP, the device is probably offline
		// and we will be not able to reestablish the connection when it will
		// come back online
		refDev.Device().Close(ctx)
		if refDev.Device().FoundByIP() == "" {
			c.deviceCache.RemoveDevice(ctx, refDev.DeviceID(), refDev)
			refDev.Release(ctx)
		}
		return nil, nil, fmt.Errorf("cannot get links for device %v: %w", deviceID, err)
	}
	return refDev, patchResourceLinksEndpoints(links, c.disableUDPEndpoints), nil
}

func (c *Client) getDevice(ctx context.Context, refDev *RefDevice, links schema.ResourceLinks, getDetails GetDetailsFunc) (DeviceDetails, error) {
	devDetails, err := refDev.GetDeviceDetails(ctx, links, getDetails)
	if err != nil {
		return DeviceDetails{}, err
	}
	var o ownership
	if devDetails.IsSecured {
		d, ownErr := refDev.GetOwnership(ctx, links)
		if ownErr != nil {
			v, ok := status.FromError(ownErr)
			if ok && v.Code() == codes.Unauthorized {
				o.status = OwnershipStatus_OwnedByOther
			}
		} else {
			o.doxm = &d
		}
	}
	ownerID, _ := c.client.GetSdkOwnerID()

	return setOwnership(ownerID, map[string]DeviceDetails{
		devDetails.ID: devDetails,
	}, map[string]ownership{
		devDetails.ID: o,
	})[devDetails.ID], nil
}

func (c *Client) GetDeviceByMulticast(ctx context.Context, deviceID string, opts ...GetDeviceOption) (DeviceDetails, error) {
	cfg := getDeviceOptions{
		getDetails:             getDetails,
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnGetDevice(cfg)
	}

	refDev, links, err := c.GetRefDevice(ctx, deviceID, opts...)
	if err != nil {
		return DeviceDetails{}, err
	}
	defer refDev.Release(ctx)
	return c.getDevice(ctx, refDev, links, cfg.getDetails)
}

func (c *Client) GetAllDeviceIDsFoundByIP() map[string]string {
	return c.deviceCache.GetContent()
}

// GetDeviceByIP gets the device directly via IP address and multicast listen port 5683.
func (c *Client) GetDeviceByIP(ctx context.Context, ip string, opts ...GetDeviceByIPOption) (DeviceDetails, error) {
	cfg := getDeviceByIPOptions{
		getDetails: getDetails,
	}
	fmt.Println("######################### client.GetDeviceByIP")
	for _, o := range opts {
		cfg = o.applyOnGetDeviceByIP(cfg)
	}

	refDev, links, err := c.GetRefDeviceByIP(ctx, ip)

	if err != nil {
		fmt.Println(err)
		return DeviceDetails{}, err
	}
	defer refDev.Release(ctx)
	return c.getDevice(ctx, refDev, links, cfg.getDetails)
}
