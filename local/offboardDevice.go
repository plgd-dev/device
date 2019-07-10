package local

import (
	"context"
	"fmt"
)

func (d *Device) OffboardInsecured(
	ctx context.Context,
) error {
	const errMsg = "cannot offboard device: %v"
	ok, err := d.IsSecured(ctx)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	if ok {
		return fmt.Errorf(errMsg, "is secured device")
	}

	err = d.onboardOffboardInsecuredDevice(ctx, "", "", "")
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

func (d *Device) Offboard(
	ctx context.Context,
) error {
	const errMsg = "cannot offboard device: %v"
	ok, err := d.IsSecured(ctx)
	if err != nil {
		return fmt.Errorf(errMsg, err)
	}

	if !ok {
		return fmt.Errorf(errMsg, "is insecured device")
	}

	return d.Disown(ctx)
}
