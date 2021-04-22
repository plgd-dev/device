package core

import (
	"context"
)

// IsSecured returns if device serves secure ports.
func (d *Device) IsSecured(ctx context.Context) (bool, error) {
	eps, err := d.GetEndpoints(ctx)
	if err != nil {
		return false, err
	}
	if len(eps.FilterSecureEndpoints()) > 0 {
		return true, nil
	}
	return false, nil
}
