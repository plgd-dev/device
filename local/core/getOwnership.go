package core

import (
	"context"
	"fmt"

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

	var ownership schema.Doxm
	err := d.GetResource(ctx, getOwnlink, &ownership)
	return ownership, err
}
