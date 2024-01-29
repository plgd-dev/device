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

package service

import (
	ocfCloud "github.com/plgd-dev/device/v2/pkg/ocf/cloud"
	"github.com/plgd-dev/go-coap/v3/tcp/client"
)

type ServiceHandlerConfig struct {
	coapConn *client.Conn
}

func (s *ServiceHandlerConfig) GetCoapConnection() *client.Conn {
	return s.coapConn
}

type Option interface {
	Apply(o *ServiceHandlerConfig)
}

type CoapConnectionOpt struct {
	coapConn *client.Conn
}

func (o CoapConnectionOpt) Apply(opts *ServiceHandlerConfig) {
	opts.coapConn = o.coapConn
}

func WithCoapConnectionOpt(c *client.Conn) CoapConnectionOpt {
	return CoapConnectionOpt{
		coapConn: c,
	}
}

type GetServiceHandler = func(service *Service, opts ...Option) ServiceHandler

type OnShutdown = func(ServiceHandler)

type ServiceHandler interface {
	CloseOnError() bool
	SignUp(req ocfCloud.CoapSignUpRequest) (ocfCloud.CoapSignUpResponse, error)
	SignOff() error
	SignIn(req ocfCloud.CoapSignInRequest) (ocfCloud.CoapSignInResponse, error)
	SignOut(req ocfCloud.CoapSignInRequest) error
	PublishResources(req ocfCloud.PublishResourcesRequest) error
	UnpublishResources(req ocfCloud.UnpublishResourcesRequest) error
	RefreshToken(req ocfCloud.CoapRefreshTokenRequest) (ocfCloud.CoapRefreshTokenResponse, error)
}
