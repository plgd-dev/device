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
	ErrMissingAuthorizationCode     = errors.New("authorization code missing")
	ErrMissingAuthorizationProvider = errors.New("authorization provider missing")
	ErrCannotSignUp                 = errors.New("cannot sign up")
)

func MakeSignUpRequest(deviceID, code, provider string) (ocfCloud.CoapSignUpRequest, error) {
	if code == "" {
		return ocfCloud.CoapSignUpRequest{}, ErrMissingAuthorizationCode
	}
	if provider == "" {
		return ocfCloud.CoapSignUpRequest{}, ErrMissingAuthorizationProvider
	}

	return ocfCloud.CoapSignUpRequest{
		DeviceID:              deviceID,
		AuthorizationCode:     code,
		AuthorizationProvider: provider,
	}, nil
}

func errCannotSignUp(err error) error {
	return fmt.Errorf("%w: %w", ErrCannotSignUp, err)
}

func (c *Manager) signUp(ctx context.Context) error {
	creds := c.getCreds()
	if creds.AccessToken != "" {
		return nil
	}
	cfg := c.getCloudConfiguration()
	signUpRequest, err := MakeSignUpRequest(c.deviceID.String(), cfg.AuthorizationCode, cfg.AuthorizationProvider)
	if err != nil {
		return errCannotSignUp(err)
	}
	client := c.getClient()
	if client == nil {
		return errCannotSignOff(errors.New("no connection"))
	}
	req, err := newPostRequest(ctx, client, ocfCloud.SignUp, signUpRequest)
	if err != nil {
		return errCannotSignUp(err)
	}
	c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERING)
	resp, err := client.Do(req)
	if err != nil {
		return errCannotSignUp(err)
	}
	if resp.Code() != codes.Changed {
		return errCannotSignUp(fmt.Errorf("unexpected status code %v", resp.Code()))
	}
	var signUpResp ocfCloud.CoapSignUpResponse
	err = cbor.ReadFrom(resp.Body(), &signUpResp)
	if err != nil {
		return errCannotSignUp(err)
	}
	signUpResp.ValidUntil = validUntil(signUpResp.ExpiresIn)
	c.setCreds(signUpResp)
	c.logger.Infof("signed up")
	c.save()
	return nil
}
