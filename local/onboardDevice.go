package local

import (
	"context"
	"fmt"

	"github.com/go-ocf/sdk/schema"
)

func (d *Device) onboardOffboardInsecuredDevice(
	ctx context.Context,
	authorizationProvider, authorizationCode, url string,
) error {
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
	err := d.UpdateResource(ctx, cloudResourceHref, req, &resp)
	if err != nil {
		return err
	}
	return nil
}

func (d *Device) IsSecured(ctx context.Context) (bool, error) {
	return d.DeviceLinks.IsSecured(), nil
}

type ProvisionDeviceFunc = func(ctx context.Context, c *ProvisioningClient) error

// Onboard onboards secure device.
func (d *Device) Onboard(
	ctx context.Context,
	otmClient OTMClient,
	provision ProvisionDeviceFunc,
) error {
	const errMsg = "cannot onboard secured device  %v"
	if otmClient == nil {
		return fmt.Errorf(errMsg, "invalid otmClient")
	}
	if provision == nil {
		return fmt.Errorf(errMsg, "invalid provision function")
	}
	ok, err := d.IsSecured(ctx)
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot determines if device is secured: %v", err))
	}
	if !ok {
		return fmt.Errorf(errMsg, fmt.Errorf("device is insecured"))
	}
	err = d.onboardSecuredDevice(ctx, otmClient, provision)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

// OnboardInsecured onboards insecure device.
func (d *Device) OnboardInsecured(ctx context.Context, authorizationProvider, authorizationCode, url string) error {
	const errMsg = "cannot onboard device: %v"
	switch {
	case authorizationProvider == "":
		return fmt.Errorf(errMsg, "invalid authorizationProvider")
	case authorizationCode == "":
		return fmt.Errorf(errMsg, "invalid authorizationCode")
	case url == "":
		return fmt.Errorf(errMsg, "invalid url")
	}
	ok, err := d.IsSecured(ctx)
	if err != nil {
		return fmt.Errorf(errMsg, fmt.Errorf("cannot determines if device is secured: %v", err))
	}
	if ok {
		return fmt.Errorf(errMsg, "is insecured device")
	}

	err = d.onboardOffboardInsecuredDevice(ctx, authorizationProvider, authorizationCode, url)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

func (d *Device) onboardSecuredDevice(ctx context.Context, otmClient OTMClient, provision ProvisionDeviceFunc) error {
	err := d.Own(ctx, otmClient)
	if err != nil {
		return err
	}

	provisionClient, err := d.Provision(ctx)
	if err != nil {
		d.Disown(ctx)
		return err
	}

	err = provision(ctx, provisionClient)
	if err != nil {
		provisionClient.Close(ctx)
		d.Disown(ctx)
		return err
	}
	err = provisionClient.Close(ctx)
	if err != nil {
		return err
	}
	return nil
}
