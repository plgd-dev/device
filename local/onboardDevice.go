package local

import (
	"context"
	"fmt"

	"github.com/go-ocf/sdk/schema"
)

func (c *Client) onboardOffboardDevice(
	ctx context.Context,
	deviceID, authorizationProvider, authorizationCode, url string, errFunc func(err error) error,
) error {
	switch {
	case deviceID == "":
		return errFunc(fmt.Errorf("invalid deviceID"))
	}

	var links []schema.ResourceLink
	err := c.GetResourceCBOR(ctx, deviceID, "/oic/res", &links)
	if err != nil {
		return errFunc(err)
	}

	cloudResourceHref := ""
Loop:
	for _, link := range links {
		for _, resType := range link.ResourceTypes {
			if resType == schema.CloudResourceType {
				cloudResourceHref = link.Href
				break Loop
			}
		}
	}

	if cloudResourceHref == "" {
		return errFunc(fmt.Errorf("cloud resource not found"))
	}

	req := schema.CloudUpdateRequest{
		AuthorizationProvider: authorizationProvider,
		AuthorizationCode:     authorizationCode,
		URL:                   url,
	}
	var resp schema.CloudResponse
	err = c.UpdateResourceCBOR(ctx, deviceID, cloudResourceHref, req, &resp)
	if err != nil {
		return errFunc(err)
	}
	return nil
}

func (c *Client) OnboardDevice(
	ctx context.Context,
	deviceID, authorizationProvider, authorizationCode, url string,
) error {
	const errMsg = "cannot onboard device %v: %v"
	switch {
	case authorizationProvider == "":
		return fmt.Errorf(errMsg, deviceID, "invalid authorizationProvider")
	case authorizationCode == "":
		return fmt.Errorf(errMsg, deviceID, "invalid authorizationCode")
	case url == "":
		return fmt.Errorf(errMsg, deviceID, "invalid url")
	}
	return c.onboardOffboardDevice(ctx, deviceID, authorizationProvider, authorizationCode, url, func(err error) error {
		return fmt.Errorf(errMsg, deviceID, err)
	})
}
