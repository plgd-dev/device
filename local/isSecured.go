package local

import "context"

// IsSecured returns if device serves secure ports.
func (d *Device) IsSecured(ctx context.Context) (bool, error) {
	for _, link := range d.links {
		if _, err := link.GetTCPSecureAddr(); err == nil {
			return true, nil
		}
		if _, err := link.GetUDPSecureAddr(); err == nil {
			return true, nil
		}
	}
	return false, nil
}
