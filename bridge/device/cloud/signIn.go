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

var (
	ErrMissingAccessToken = errors.New("access token missing")
	ErrCannotSignIn       = errors.New("cannot sign in")
)

func MakeSignInRequest(deviceID, userID, accessToken string) (ocfCloud.CoapSignInRequest, error) {
	if accessToken == "" {
		return ocfCloud.CoapSignInRequest{}, ErrMissingAccessToken
	}
	return ocfCloud.CoapSignInRequest{
		DeviceID:    deviceID,
		UserID:      userID,
		AccessToken: accessToken,
		Login:       true,
	}, nil
}

func errCannotSignIn(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotSignIn, err)
}

func (c *Manager) signIn(ctx context.Context) error {
	client := c.getClient()
	if client == nil {
		return errCannotSignIn(errors.New("no connection"))
	}
	if c.isSignedIn() {
		return nil
	}
	creds := c.getCreds()
	signInReq, err := MakeSignInRequest(c.deviceID.String(), creds.UserID, creds.AccessToken)
	if err != nil {
		return errCannotSignIn(err)
	}
	req, err := newPostRequest(ctx, client, ocfCloud.SignIn, signInReq)
	if err != nil {
		return errCannotSignIn(err)
	}
	c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERING)
	resp, err := client.Do(req)
	if err != nil {
		return errCannotSignIn(err)
	}
	if resp.Code() != codes.Changed {
		if resp.Code() == codes.Unauthorized {
			if creds.RefreshToken == "" {
				c.cleanup()
			} else {
				c.forceRefreshToken = true
			}
		}
		return errCannotSignIn(fmt.Errorf("unexpected status code %v", resp.Code()))
	}
	var signInResp ocfCloud.CoapSignInResponse
	err = cbor.ReadFrom(resp.Body(), &signInResp)
	if err != nil {
		return errCannotSignIn(err)
	}
	c.updateCredsBySignInResponse(signInResp)
	c.logger.Infof("signed in")
	c.save()
	return nil
}

func (c *Manager) updateCredsBySignInResponse(resp ocfCloud.CoapSignInResponse) {
	c.private.mutex.Lock()
	defer c.private.mutex.Unlock()
	c.private.creds.ExpiresIn = resp.ExpiresIn
	c.private.creds.ValidUntil = validUntil(resp.ExpiresIn)
	c.private.signedIn = true
}
