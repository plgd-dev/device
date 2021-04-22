package core

import (
	"context"

	"github.com/plgd-dev/sdk/schema"
)

func (d *Device) GetEndpoints(ctx context.Context) (schema.Endpoints, error) {
	return d.endpoints, nil
}
