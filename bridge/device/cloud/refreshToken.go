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

	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/go-coap/v3/message/codes"
)

type CoapRefreshTokenRequest struct {
	DeviceID     string `json:"di"`
	UserID       string `json:"uid"`
	RefreshToken string `json:"refreshtoken"`
}

type CoapRefreshTokenResponse struct {
	AccessToken  string `json:"accesstoken"`
	RefreshToken string `json:"refreshtoken"`
	ExpiresIn    int64  `json:"expiresin"`
}

var ErrCannotRefreshToken = fmt.Errorf("cannot refresh token")

func errCannotRefreshToken(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotRefreshToken, err)
}

func (c *Manager) refreshToken(ctx context.Context) error {
	creds := c.getCreds()
	if creds.RefreshToken == "" {
		return nil
	}
	req, err := newPostRequest(ctx, c.client, RefreshToken, CoapRefreshTokenRequest{
		DeviceID:     c.deviceID.String(),
		UserID:       creds.UserID,
		RefreshToken: creds.RefreshToken,
	})
	if err != nil {
		return errCannotRefreshToken(err)
	}
	c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERING)
	resp, err := c.client.Do(req)
	if err != nil {
		return errCannotRefreshToken(err)
	}
	if resp.Code() != codes.Changed {
		if resp.Code() == codes.Unauthorized {
			c.cleanup()
		}
		return errCannotRefreshToken(fmt.Errorf("unexpected status code %v", resp.Code()))
	}
	var refreshResp CoapRefreshTokenResponse
	if err = cbor.ReadFrom(resp.Body(), &refreshResp); err != nil {
		return errCannotRefreshToken(err)
	}
	c.updateCredsByRefreshTokenResponse(refreshResp)
	log.Printf("refresh token\n")
	c.save()
	return nil
}

func (c *Manager) updateCredsByRefreshTokenResponse(resp CoapRefreshTokenResponse) {
	c.creds.AccessToken = resp.AccessToken
	c.creds.RefreshToken = resp.RefreshToken
	c.creds.ValidUntil = validUntil(resp.ExpiresIn)
	c.signedIn = false
}
