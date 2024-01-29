/****************************************************************************
 *
 * Copyright (c) 2024 plgd.dev s.r.o.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"),
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package cloud

import (
	"context"
	"fmt"
	"log"

	ocfCloud "github.com/plgd-dev/device/v2/pkg/ocf/cloud"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/tcp/client"
)

const ProvisioningStatusDEREGISTERING cloud.ProvisioningStatus = "deregistering"

var ErrCannotSignOff = fmt.Errorf("cannot sign off")

func newSignOffReq(ctx context.Context, c *client.Conn, deviceID, userID string) (*pool.Message, error) {
	req, err := newRequestWithToken(ctx, c, ocfCloud.SignUp)
	if err != nil {
		return nil, err
	}
	req.SetCode(codes.DELETE)
	req.AddQuery("di=" + deviceID)
	req.AddQuery("uid=" + userID)
	return req, nil
}

func errCannotSignOff(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotSignOff, err)
}

func (c *Manager) signOff(ctx context.Context) error {
	if c.client == nil {
		return nil
	}
	// signIn / refresh token fails
	if ctx.Err() != nil {
		return errCannotSignOff(ctx.Err())
	}

	req, err := newSignOffReq(ctx, c.client, c.deviceID.String(), c.getCreds().UserID)
	if err != nil {
		return errCannotSignOff(err)
	}
	c.setProvisioningStatus(ProvisioningStatusDEREGISTERING)
	resp, err := c.client.Do(req)
	defer c.setProvisioningStatus(cloud.ProvisioningStatus_UNINITIALIZED)
	if err != nil {
		return errCannotSignOff(err)
	}
	if resp.Code() != codes.Deleted {
		return errCannotSignOff(fmt.Errorf("unexpected status code %v", resp.Code()))
	}
	log.Printf("signed off\n")
	return nil
}
