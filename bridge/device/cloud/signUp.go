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
	"time"

	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	"github.com/plgd-dev/device/v2/schema/cloud"
	"github.com/plgd-dev/go-coap/v3/message/codes"
)

type CoapSignUpRequest struct {
	DeviceID              string `json:"di"`
	AuthorizationCode     string `json:"accesstoken"`
	AuthorizationProvider string `json:"authprovider"`
}

type CoapSignUpResponse struct {
	AccessToken  string    `yaml:"accessToken" json:"accesstoken"`
	UserID       string    `yaml:"userID" json:"uid"`
	RefreshToken string    `yaml:"refreshToken" json:"refreshtoken"`
	RedirectURI  string    `yaml:"-" json:"redirecturi"`
	ExpiresIn    int64     `yaml:"-" json:"expiresin"`
	ValidUntil   time.Time `yaml:"-" jsom:"-"`
}

var (
	ErrMissingAuthorizationCode     = fmt.Errorf("authorization code missing")
	ErrMissingAuthorizationProvider = fmt.Errorf("authorization provider missing")
	ErrCannotSignUp                 = fmt.Errorf("cannot sign up")
)

func MakeSignUpRequest(deviceID, code, provider string) (CoapSignUpRequest, error) {
	if code == "" {
		return CoapSignUpRequest{}, ErrMissingAuthorizationCode
	}
	if provider == "" {
		return CoapSignUpRequest{}, ErrMissingAuthorizationProvider
	}

	return CoapSignUpRequest{
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
	req, err := newPostRequest(ctx, c.client, SignUp, signUpRequest)
	if err != nil {
		return errCannotSignUp(err)
	}
	c.setProvisioningStatus(cloud.ProvisioningStatus_REGISTERING)
	resp, err := c.client.Do(req)
	if err != nil {
		return errCannotSignUp(err)
	}
	if resp.Code() != codes.Changed {
		return errCannotSignUp(fmt.Errorf("unexpected status code %v", resp.Code()))
	}
	var signUpResp CoapSignUpResponse
	err = cbor.ReadFrom(resp.Body(), &signUpResp)
	if err != nil {
		return errCannotSignUp(err)
	}
	if signUpResp.ExpiresIn != -1 {
		c.creds.ValidUntil = validUntil(signUpResp.ExpiresIn)
	}
	c.setCreds(signUpResp)
	log.Printf("signed up\n")
	c.save()
	return nil
}
