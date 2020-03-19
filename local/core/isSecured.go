package core

import (
	"context"

	"github.com/go-ocf/sdk/schema"
)

// IsSecured returns if device serves secure ports.
func (d *Device) IsSecured(ctx context.Context, links schema.ResourceLinks) (bool, error) {
	for _, link := range links {
		if _, err := link.GetTCPSecureAddr(); err == nil {
			return true, nil
		}
		if _, err := link.GetUDPSecureAddr(); err == nil {
			return true, nil
		}
	}
	return false, nil
}
