package local

import (
	"context"
	"fmt"

	"github.com/go-ocf/sdk/schema"
)

func (c *Client) ProvisionDevice(deviceID string) (*ProvisioningClient, error) {
	p := ProvisioningClient{Client: c, deviceID: deviceID}
	err := p.start()
	if err != nil {
		return nil, err
	}
	return &p, nil
}

type ProvisioningClient struct {
	*Client
	deviceID string
}

func (c *ProvisioningClient) start() error {
	provisioningState := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RFPRO,
		},
	}
	err := c.UpdateResource(context.Background(), c.deviceID, "/oic/sec/pstat", provisioningState, nil)
	if err != nil {
		return fmt.Errorf("could not start provisioning the device %s: %v", c.deviceID, err)
	}
	return nil
}

func (c *ProvisioningClient) Close() error {
	normalOperationState := schema.ProvisionStatusUpdateRequest{
		DeviceOnboardingState: &schema.DeviceOnboardingState{
			CurrentOrPendingOperationalState: schema.OperationalState_RFNOP,
		},
	}
	err := c.UpdateResource(context.Background(), c.deviceID, "/oic/sec/pstat", normalOperationState, nil)
	if err != nil {
		return fmt.Errorf("could not finalize provisioning the device %s: %v", c.deviceID, err)
	}
	return nil
}
