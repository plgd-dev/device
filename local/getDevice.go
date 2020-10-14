package local

import (
	"context"

	"github.com/plgd-dev/sdk/local/core"
	"github.com/plgd-dev/sdk/schema"
)

// GetRefDevice returns device, after using call device.Release to free resources.
func (c *Client) GetRefDevice(
	ctx context.Context,
	deviceID string,
) (*RefDevice, schema.ResourceLinks, error) {
	refDev, ok := c.deviceCache.GetDevice(ctx, deviceID)
	if ok {
		endpoints, err := refDev.GetEndpoints(ctx)
		if err != nil {
			if err != nil {
				return nil, nil, err
			}
		}
		links, err := refDev.GetResourceLinks(ctx, endpoints)
		if err != nil {
			return nil, nil, err
		}
		return refDev, c.PatchResourceLinksEndpoints(links), nil
	}
	dev, links, err := c.client.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, nil, err
	}

	newRefDev := NewRefDevice(dev)
	refDev, stored, err := c.deviceCache.TryStoreDeviceToTemporaryCache(newRefDev)
	if !stored {
		newRefDev.Release(ctx)
	}
	if err != nil {
		return nil, nil, err
	}
	return refDev, c.PatchResourceLinksEndpoints(links), nil
}

func (c *Client) GetDevice(ctx context.Context, deviceID string, opts ...GetDeviceOption) (DeviceDetails, error) {
	cfg := getDeviceOptions{
		getDetails: func(context.Context, *core.Device, schema.ResourceLinks) (interface{}, error) {
			return nil, nil
		},
	}
	for _, o := range opts {
		cfg = o.applyOnGetDevice(cfg)
	}

	refDev, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return DeviceDetails{}, err
	}
	defer refDev.Release(ctx)

	devDetails, err := refDev.GetDeviceDetails(ctx, links, cfg.getDetails)
	if err != nil {
		return DeviceDetails{}, err
	}
	var doxm schema.Doxm
	if devDetails.IsSecured {
		doxm, err = refDev.GetOwnership(ctx)
	}
	if err != nil {
		return DeviceDetails{}, err
	}
	ownerID, _ := c.client.GetSdkOwnerID()

	return setOwnership(ownerID, map[string]DeviceDetails{
		devDetails.Device.ID: devDetails,
	}, map[string]schema.Doxm{
		doxm.DeviceID: doxm,
	})[devDetails.Device.ID], nil
}
