package client

import (
	"context"

	pkgError "github.com/plgd-dev/device/pkg/error"
)

// OffboardDevice is not supported by OCF spec(https://openconnectivity.org/specs/OCF_Device_To_Cloud_Services_Specification_v2.2.0.pdf)
func (c *Client) OffboardDevice(ctx context.Context, deviceID string) error {
	return pkgError.NotSupported()
}
