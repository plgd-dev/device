package backend

import (
	"context"

	"github.com/go-ocf/grpc-gateway/pb"
	"github.com/go-ocf/sdk/schema"
)

type DeviceDetails struct {
	ID        string
	Device    pb.Device
	DeviceRaw []byte
	Resources []pb.ResourceLink
}

// GetDevices retrieves device details from the backend.
func (c *Client) GetDevices(
	ctx context.Context,
	opts ...GetDevicesOption,
) (map[string]DeviceDetails, error) {
	var cfg getDevicesOptions
	for _, o := range opts {
		cfg = o.applyOnGetDevices(cfg)
	}

	devices := make(map[string]DeviceDetails, len(cfg.deviceIDs))
	ids := make([]string, 0, len(cfg.deviceIDs))

	err := c.GetDevicesViaCallback(ctx, cfg.deviceIDs, cfg.resourceTypes, func(v pb.Device) {
		devices[v.GetId()] = DeviceDetails{
			ID:     v.GetId(),
			Device: v,
		}
		ids = append(ids, v.GetId())
	})
	if err != nil {
		return nil, err
	}

	err = c.RetrieveResourcesByType(ctx, ids,
		MakeTypeCallback(schema.DeviceResourceType, func(v pb.ResourceValue) {
			d, ok := devices[v.GetResourceId().GetDeviceId()]
			if ok && err == nil {
				d.DeviceRaw = v.GetContent().GetData()
				devices[v.GetResourceId().GetDeviceId()] = d
			}
		}),
	)
	if err != nil {
		return nil, err
	}

	err = c.GetResourceLinksViaCallback(ctx, ids, nil, func(v pb.ResourceLink) {
		d, ok := devices[v.GetDeviceId()]
		if ok {
			d.Resources = append(d.Resources, v)
			devices[v.GetDeviceId()] = d
		}
	})
	if err != nil {
		return nil, err
	}

	return devices, nil
}
