package local

import (
	"context"
	"fmt"
	"time"

	"github.com/go-ocf/sdk/local/device"
	"github.com/go-ocf/sdk/schema"
)

func (c *Client) ownDeviceFindClient(ctx context.Context, deviceID string, discoveryTimeout time.Duration, owned bool) (*device.Client, error) {
	ctxOwn, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()
	h := newDeviceHandler(deviceID, cancel)

	err := c.GetDeviceOwnership(ctxOwn, false, h)
	client := h.Client()

	if client != nil {
		return client, nil
	}
	if err != nil {
		return nil, err
	}

	return nil, h.Err()
}

func (c *Client) getDeviceLinks(ctx context.Context,
	deviceID string,
	discoveryTimeout time.Duration,
) (res schema.DeviceLinks, _ error) {
	ctxGet, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()

	var device schema.DeviceLinks
	err := c.GetResourceCBOR(ctxGet, deviceID, "/oic/res", "", &device)
	if err != nil {
		return res, err
	}
	return device, nil
}

// OwnDevice set ownership of device
func (c *Client) OwnDevice(
	ctx context.Context,
	deviceID string,
	otm schema.OwnerTransferMethod,
	discoveryTimeout time.Duration,
) error {
	const errMsg = "cannot own device %v: %v"

	client, err := c.ownDeviceFindClient(ctx, deviceID, discoveryTimeout, false)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}
	if client == nil {
		client, err = c.ownDeviceFindClient(ctx, deviceID, discoveryTimeout, true)
		if err != nil {
			return fmt.Errorf(errMsg, deviceID, err)
		}
		if client != nil {
			return nil
		}
		return fmt.Errorf(errMsg, deviceID, "not found")
	}
	ownership := client.GetOwnerShip()
	var supportOtm bool
	for _, s := range ownership.GetSupportedOwnerTransferMethods() {
		if s == otm {
			supportOtm = true
		}
		break
	}
	if !supportOtm {
		fmt.Println(fmt.Errorf(errMsg, deviceID, fmt.Sprintf("ownership transfer method '%v' is unsupported, supported are: %v", otm, ownership.GetSupportedOwnerTransferMethods())))
	}

	_, err = c.getDeviceLinks(ctx, deviceID, discoveryTimeout)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Sprintf("cannot get resource links %v:", err))
	}

	//fmt.Println(deviceLinks)

	req := schema.DoxmSelectOwnerTransferMethod{
		SelectOwnerTransferMethod: 0,
	}
	var resp schema.Doxm

	err = c.UpdateResourceCBOR(ctx, deviceID, "/oic/sec/doxm", "", req, &resp)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}

	//fmt.Println(resp)

	return nil
}
