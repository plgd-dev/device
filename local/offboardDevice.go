package local

import (
	"context"
	"fmt"
)

// OffboardDevice is not supported by OCF spec(https://openconnectivity.org/specs/OCF_Device_To_Cloud_Services_Specification_v2.2.0.pdf)
func (c *Client) OffboardDevice(ctx context.Context, deviceID string) error {
	return fmt.Errorf("not supported")
}
