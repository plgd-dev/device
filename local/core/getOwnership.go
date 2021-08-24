package core

import (
	"context"
	"fmt"

	"github.com/plgd-dev/sdk/pkg/net/coap"
	"github.com/plgd-dev/sdk/schema"
)

// GetOwnership gets device's ownership resource.
func (d *Device) GetOwnership(ctx context.Context, links schema.ResourceLinks) (schema.Doxm, error) {
	ownLink, ok := links.GetResourceLink(schema.DoxmHref)
	if !ok {
		return schema.Doxm{}, fmt.Errorf("not found")
	}
	getOwnlink := ownLink
	getOwnlink.Endpoints = ownLink.GetUnsecureEndpoints()
	if len(getOwnlink.Endpoints) == 0 {
		getOwnlink.Endpoints = ownLink.GetSecureEndpoints()
	}

	var ownership schema.Doxm
	err := d.GetResource(ctx, getOwnlink, &ownership, coap.WithInterface("oic.if.baseline"))
	return ownership, err
}
