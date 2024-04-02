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
	"time"

	"github.com/plgd-dev/device/v2/schema"
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
	ValidUntil   time.Time `yaml:"-" json:"-"`
}

type CoapSignInRequest struct {
	DeviceID    string `json:"di"`
	UserID      string `json:"uid"`
	AccessToken string `json:"accesstoken"`
	Login       bool   `json:"login"`
}

type CoapSignInResponse struct {
	ExpiresIn int64 `json:"expiresin"`
}

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

type PublishResourcesRequest struct {
	DeviceID   string               `json:"di"`
	Links      schema.ResourceLinks `json:"links"`
	TimeToLive int                  `json:"ttl"`
}

type UnpublishResourcesRequest struct {
	DeviceID    string
	InstanceIDs []int64
}
