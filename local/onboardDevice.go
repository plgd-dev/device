package local

import (
	"context"
	"fmt"

	"github.com/go-ocf/sdk/schema"
)

func (c *Client) onboardOffboardInsecuredDevice(
	ctx context.Context,
	deviceID, authorizationProvider, authorizationCode, url string,
) error {
	switch {
	case deviceID == "":
		return fmt.Errorf("invalid deviceID")
	}

	d, err := c.GetDevice(ctx, deviceID)
	if err != nil {
		return err
	}

	cloudResourceHref := ""
Loop:
	for _, link := range d.GetResourceLinks() {
		for _, resType := range link.ResourceTypes {
			if resType == schema.CloudResourceType {
				cloudResourceHref = link.Href
				break Loop
			}
		}
	}

	if cloudResourceHref == "" {
		return fmt.Errorf("cloud resource not found")
	}

	req := schema.CloudUpdateRequest{
		AuthorizationProvider: authorizationProvider,
		AuthorizationCode:     authorizationCode,
		URL:                   url,
	}
	var resp schema.CloudResponse
	err = c.UpdateResource(ctx, deviceID, cloudResourceHref, req, &resp)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) isSecuredDevice(ctx context.Context, deviceID string) (bool, error) {
	device, err := c.GetDevice(ctx, deviceID)
	if err != nil {
		return false, err
	}
	defer device.Close()
	return device.GetDeviceLinks().IsSecured(), nil
}

type ProvisionDeviceFunc = func(ctx context.Context, c *ProvisioningClient) error

// OnboardDevice onboards secure device.
func (c *Client) OnboardDevice(
	ctx context.Context,
	deviceID string,
	otmClient OTMClient,
	provision ProvisionDeviceFunc,
) error {
	const errMsg = "cannot onboard secured device %v: %v"
	if deviceID == "" {
		return fmt.Errorf(errMsg, deviceID, "invalid deviceID")
	}
	if otmClient == nil {
		return fmt.Errorf(errMsg, deviceID, "invalid otmClient")
	}
	if provision == nil {
		return fmt.Errorf(errMsg, deviceID, "invalid provision function")
	}
	ok, err := c.isSecuredDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot determines if device is secured: %v", err))
	}
	if !ok {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("device is insecured"))
	}
	err = c.onboardSecuredDevice(ctx, deviceID, otmClient, provision)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}
	return nil
}

// OnboardInsecuredDevice onboards insecure device.
func (c *Client) OnboardInsecuredDevice(ctx context.Context, deviceID, authorizationProvider, authorizationCode, url string) error {
	const errMsg = "cannot onboard device %v: %v"
	switch {
	case authorizationProvider == "":
		return fmt.Errorf(errMsg, deviceID, "invalid authorizationProvider")
	case authorizationCode == "":
		return fmt.Errorf(errMsg, deviceID, "invalid authorizationCode")
	case url == "":
		return fmt.Errorf(errMsg, deviceID, "invalid url")
	}
	ok, err := c.isSecuredDevice(ctx, deviceID)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, fmt.Errorf("cannot determines if device is secured: %v", err))
	}
	if ok {
		return fmt.Errorf(errMsg, deviceID, "is insecured device")
	}

	err = c.onboardOffboardInsecuredDevice(ctx, deviceID, authorizationProvider, authorizationCode, url)
	if err != nil {
		return fmt.Errorf(errMsg, deviceID, err)
	}
	return nil
}

func (c *Client) onboardSecuredDevice(ctx context.Context, deviceID string, otmClient OTMClient, provision ProvisionDeviceFunc) error {
	err := c.OwnDevice(ctx, deviceID, otmClient)
	if err != nil {
		return err
	}

	provisionClient, err := c.ProvisionDevice(ctx, deviceID)
	if err != nil {
		c.DisownDevice(ctx, deviceID)
		return err
	}

	err = provision(ctx, provisionClient)
	if err != nil {
		provisionClient.Close(ctx)
		c.DisownDevice(ctx, deviceID)
		return err
	}
	err = provisionClient.Close(ctx)
	if err != nil {
		return err
	}
	return nil
}
