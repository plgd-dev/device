package local

import (
	"context"

	ocf "github.com/go-ocf/sdk/local/core"
	ocfschema "github.com/go-ocf/sdk/schema"
)

// GetRefDevice returns device, after using call device.Release to free resources.
func (c *Client) GetRefDevice(
	ctx context.Context,
	deviceID string,
) (*RefDevice, ocfschema.ResourceLinks, error) {
	refDev, ok := c.deviceCache.GetDevice(ctx, deviceID)
	if ok {
		links, err := refDev.GetResourceLinks(ctx)
		if err != nil {
			return nil, nil, err
		}
		return refDev, c.patchResourceLinks(links), nil
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
	return refDev, c.patchResourceLinks(links), nil
}

func (c *Client) GetDevice(ctx context.Context, deviceID string, opts ...GetDeviceOption) (DeviceDetails, error) {
	cfg := getDeviceOptions{
		getDetails: func(context.Context, *ocf.Device, ocfschema.ResourceLinks) (interface{}, error) {
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
	var doxm ocfschema.Doxm
	if devDetails.IsSecured {
		doxm, err = refDev.GetOwnership(ctx)
	}
	if err != nil {
		return DeviceDetails{}, err
	}

	return setOwnership(map[string]DeviceDetails{
		devDetails.Device.ID: devDetails,
	}, map[string]ocfschema.Doxm{
		doxm.DeviceID: doxm,
	})[devDetails.Device.ID], nil
}
