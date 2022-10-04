package client

import (
	"context"

	"github.com/plgd-dev/device/client/core"
	"github.com/plgd-dev/device/client/core/otm"
)

func (c *Client) OwnDevice(ctx context.Context, deviceID string, opts ...OwnOption) (string, error) {
	cfg := ownOptions{
		otmTypes:               []OTMType{OTMType_JustWorks},
		discoveryConfiguration: core.DefaultDiscoveryConfiguration(),
	}
	for _, o := range opts {
		cfg = o.applyOnOwn(cfg)
	}
	d, _, err := c.GetRefDevice(ctx, deviceID, WithDiscoveryConfiguration(cfg.discoveryConfiguration))
	if err != nil {
		return "", err
	}
	defer d.Release(ctx)
	ok := d.IsSecured()
	if err != nil {
		return "", err
	}
	if !ok {
		// don't own insecure device
		return deviceID, nil
	}
	return c.deviceOwner.OwnDevice(ctx, deviceID, cfg.otmTypes, cfg.discoveryConfiguration, c.ownDeviceWithSigners, cfg.opts...)
}

func (c *Client) updateCache(ctx context.Context, d *RefDevice, deviceID string) {
	if d.DeviceID() != deviceID {
		if c.deviceCache.RemoveDevice(deviceID, d) {
			for {
				storedDev, stored := c.deviceCache.TryStoreDevice(d)
				if stored {
					break
				}
				c.deviceCache.RemoveDevice(storedDev.DeviceID(), storedDev)
				storedDev.Release(ctx)
			}
		}
	}
}

func (c *Client) ownDeviceWithSigners(ctx context.Context, deviceID string, otmClient []otm.Client, discoveryConfiguration core.DiscoveryConfiguration, opts ...core.OwnOption) (string, error) {
	d, links, err := c.GetRefDevice(ctx, deviceID, WithDiscoveryConfiguration(discoveryConfiguration))
	if err != nil {
		return "", err
	}
	defer d.Release(ctx)
	ok := d.IsSecured()
	if !ok {
		// don't own insecure device
		return d.DeviceID(), nil
	}
	if c.disableUDPEndpoints {
		// we need to get all links because just-works need to use dtls
		endpoints := d.GetEndpoints()
		links, err = d.GetResourceLinks(ctx, endpoints)
		if err != nil {
			return "", err
		}
		links = patchResourceLinksEndpoints(links, false)
	}

	err = d.Own(ctx, links, otmClient, opts...)
	if err != nil {
		return "", err
	}
	c.updateCache(ctx, d, deviceID)

	return d.DeviceID(), nil
}
