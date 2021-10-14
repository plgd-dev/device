package core

import (
	"context"
	"fmt"

	"github.com/plgd-dev/device/pkg/net/coap"
	"github.com/plgd-dev/device/schema"
	"github.com/plgd-dev/device/schema/doxm"
	"github.com/plgd-dev/device/schema/interfaces"
)

// GetOwnership gets device's ownership resource.
func (d *Device) GetOwnership(ctx context.Context, links schema.ResourceLinks) (doxm.Doxm, error) {
	ownLink, ok := links.GetResourceLink(doxm.ResourceURI)
	if !ok {
		return doxm.Doxm{}, fmt.Errorf("not found")
	}
	getOwnlink := ownLink
	getOwnlink.Endpoints = ownLink.GetUnsecureEndpoints()
	if len(getOwnlink.Endpoints) == 0 {
		getOwnlink.Endpoints = ownLink.GetSecureEndpoints()
	}

	var ownership doxm.Doxm
	err := d.GetResource(ctx, getOwnlink, &ownership, coap.WithInterface(interfaces.OC_IF_BASELINE))
	return ownership, err
}
