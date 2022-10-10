package client

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/v2/client/core"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/status"
)

func getLinksDevice(ctx context.Context, dev *core.Device, disableUDPEndpoints bool) (schema.ResourceLinks, error) {
	endpoints := dev.GetEndpoints()
	links, err := dev.GetResourceLinks(ctx, endpoints)
	if err != nil {
		return nil, err
	}
	return patchResourceLinksEndpoints(links, disableUDPEndpoints), nil
}

// Don't remove devices found by IP, the device is probably offline
// and we will be not able to reestablish the connection when it will
// come back online
func deleteDeviceNotFoundByIP(ctx context.Context, deviceCache *DeviceCache, dev *core.Device) {
	if dev.FoundByIP() == "" {
		deviceCache.LoadAndDeleteDevice(ctx, dev.DeviceID())
	}
	dev.Close(ctx)
}

func getDeviceFromCache(ctx context.Context, deviceCache *DeviceCache,
	deviceID string, disableUDPEndpoints bool,
) (*core.Device, schema.ResourceLinks, bool) {
	dev, ok := deviceCache.GetDevice(deviceID)
	if ok {
		links, err := getLinksDevice(ctx, dev, disableUDPEndpoints)
		if err != nil {
			deleteDeviceNotFoundByIP(ctx, deviceCache, dev)
			return nil, nil, false
		}
		return dev, links, true
	}
	return nil, nil, false
}

// GetDeviceByIP gets the device directly via IP address and multicast listen port 5683.
func (c *Client) GetDeviceByIP(
	ctx context.Context,
	ip string,
) (*core.Device, schema.ResourceLinks, error) {
	// we are intentionally not searching for the device inside the cache
	// as we wan't to contact the device
	newDev, err := c.client.GetDeviceByIP(ctx, ip)
	if err != nil {
		if dev := c.deviceCache.GetDeviceByFoundIP(ip); dev != nil {
			if dev.IsConnected() {
				dev.Close(ctx)
			}
		}
		return nil, nil, err
	}

	dev, _ := c.deviceCache.UpdateOrStoreDevice(newDev)
	links, err := getLinksDevice(ctx, dev, c.disableUDPEndpoints)
	if err != nil {
		deviceID := dev.DeviceID()
		deleteDeviceNotFoundByIP(ctx, c.deviceCache, dev)
		return nil, nil, fmt.Errorf("cannot get links for device %v: %w", deviceID, err)
	}
	return dev, patchResourceLinksEndpoints(links, c.disableUDPEndpoints), nil
}

// GetDeviceByMulticast returns device.
func (c *Client) GetDeviceByMulticast(
	ctx context.Context,
	deviceID string,
	opts ...GetDeviceOption,
) (*core.Device, schema.ResourceLinks, error) {
	cfg := getDeviceOptions{
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnGetDevice(cfg)
	}
	dev, links, ok := getDeviceFromCache(ctx, c.deviceCache, deviceID, c.disableUDPEndpoints)
	if ok {
		return dev, links, nil
	}
	newdev, err := c.client.GetDeviceByMulticast(ctx, deviceID, cfg.discoveryConfiguration)
	if err != nil {
		return nil, nil, err
	}

	dev, _ = c.deviceCache.UpdateOrStoreDeviceWithExpiration(newdev)
	links, err = getLinksDevice(ctx, dev, c.disableUDPEndpoints)
	if err != nil {
		deleteDeviceNotFoundByIP(ctx, c.deviceCache, dev)
		return nil, nil, fmt.Errorf("cannot get links for device %v: %w", deviceID, err)
	}
	return dev, patchResourceLinksEndpoints(links, c.disableUDPEndpoints), nil
}

func (c *Client) getDevice(ctx context.Context, dev *core.Device, links schema.ResourceLinks, getDetails GetDetailsFunc) (DeviceDetails, error) {
	devDetails, err := getDeviceDetails(ctx, dev, links, getDetails)
	if err != nil {
		return DeviceDetails{}, err
	}
	var o ownership
	if devDetails.IsSecured {
		d, ownErr := dev.GetOwnership(ctx, links)
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

func (c *Client) GetDeviceDetailsByMulticast(ctx context.Context, deviceID string, opts ...GetDeviceOption) (DeviceDetails, error) {
	cfg := getDeviceOptions{
		getDetails:             getDetails,
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnGetDevice(cfg)
	}

	dev, links, err := c.GetDeviceByMulticast(ctx, deviceID, opts...)
	if err != nil {
		return DeviceDetails{}, err
	}
	return c.getDevice(ctx, dev, links, cfg.getDetails)
}

func (c *Client) GetAllDeviceIDsFoundByIP() map[string]string {
	return c.deviceCache.GetDevicesFoundByIP()
}

// GetDeviceDetailsByIP gets the device directly via IP address and multicast listen port 5683.
func (c *Client) GetDeviceDetailsByIP(ctx context.Context, ip string, opts ...GetDeviceByIPOption) (DeviceDetails, error) {
	cfg := getDeviceByIPOptions{
		getDetails: getDetails,
	}
	for _, o := range opts {
		cfg = o.applyOnGetDeviceByIP(cfg)
	}

	dev, links, err := c.GetDeviceByIP(ctx, ip)
	if err != nil {
		return DeviceDetails{}, err
	}
	return c.getDevice(ctx, dev, links, cfg.getDetails)
}
