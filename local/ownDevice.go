package local

import (
	"context"

	"github.com/plgd-dev/sdk/local/core"
)

func (c *Client) OwnDevice(ctx context.Context, deviceID string, opts ...OwnOption) (string, error) {
	cfg := ownOptions{
		otmType: OTMType_Manufacturer,
	}
	for _, o := range opts {
		cfg = o.applyOnOwn(cfg)
	}
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return "", err
	}
	defer d.Release(ctx)
	ok, err := d.IsSecured(ctx, links)
	if err != nil {
		return "", err
	}
	if !ok {
		// don't own insecure device
		return deviceID, nil
	}

	return c.deviceOwner.OwnDevice(ctx, deviceID, cfg.otmType, c.ownDeviceWithSigners, cfg.opts...)
}

func (c *Client) ownDeviceWithSigners(ctx context.Context, deviceID string, otmClient core.OTMClient, opts ...core.OwnOption) (string, error) {
	d, links, err := c.GetRefDevice(ctx, deviceID)
	if err != nil {
		return "", err
	}
	defer d.Release(ctx)
	ok, err := d.IsSecured(ctx, links)
	if err != nil {
		return "", err
	}
	if !ok {
		// don't own insecure device
		return d.DeviceID(), nil
	}
	if c.disableUDPEndpoints {
		// we need to get all links because just-works need to use dtls
		endpoints, err := d.GetEndpoints(ctx)
		if err != nil {
			return "", err
		}
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

	return d.DeviceID(), nil
}
