package core

// IsSecured returns if device serves secure ports.
func (d *Device) IsSecured() bool {
	eps := d.GetEndpoints()
	return len(eps.FilterSecureEndpoints()) > 0
}
