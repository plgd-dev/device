package core

// IsSecured returns if device serves secure ports.
func (d *Device) IsSecured() bool {
	eps := d.GetEndpoints()
	if len(eps.FilterSecureEndpoints()) > 0 {
		return true
	}
	return false
}
