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
	"errors"
	"fmt"

	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	ocfCloud "github.com/plgd-dev/device/v2/pkg/ocf/cloud"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/go-coap/v3/message/codes"
)

var ErrCannotRefreshToken = errors.New("cannot refresh token")

func errCannotRefreshToken(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotRefreshToken, err)
}

func (c *Manager) refreshToken(ctx context.Context) error {
	creds := c.getCreds()
	if creds.RefreshToken == "" {
		return nil
	}
	client := c.getClient()
	if client == nil {
		return errCannotRefreshToken(errors.New("no connection"))
	}
	req, err := newPostRequest(ctx, client, ocfCloud.RefreshToken, ocfCloud.CoapRefreshTokenRequest{
		DeviceID:     c.deviceID.String(),
		UserID:       creds.UserID,
		RefreshToken: creds.RefreshToken,
	})
	if err != nil {
		return errCannotRefreshToken(err)
	}
	c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERING)
	resp, err := client.Do(req)
	if err != nil {
		return errCannotRefreshToken(err)
	}
	if resp.Code() != codes.Changed {
		if resp.Code() == codes.Unauthorized {
			c.cleanup()
		}
		return errCannotRefreshToken(fmt.Errorf("unexpected status code %v", resp.Code()))
	}
	var refreshResp ocfCloud.CoapRefreshTokenResponse
	if err = cbor.ReadFrom(resp.Body(), &refreshResp); err != nil {
		return errCannotRefreshToken(err)
	}
	c.updateCredsByRefreshTokenResponse(refreshResp)
	c.logger.Infof("refreshed token")
	c.save()
	return nil
}

func (c *Manager) updateCredsByRefreshTokenResponse(resp ocfCloud.CoapRefreshTokenResponse) {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	c.private.creds.AccessToken = resp.AccessToken
	c.private.creds.RefreshToken = resp.RefreshToken
	c.private.creds.ValidUntil = validUntil(resp.ExpiresIn)
	c.private.signedIn = false
}
