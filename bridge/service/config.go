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
	"errors"
	"fmt"

	"github.com/plgd-dev/device/v2/bridge/net"
)

type CoAPConfig struct {
	ID         string `yaml:"id"`
	net.Config `yaml:",inline"`
}

func (c *CoAPConfig) Validate() error {
	if c.ID == "" {
		return errors.New("id is required")
	}
	if err := c.Config.Validate(); err != nil {
		return err
	}
	return nil
}

type APIConfig struct {
	CoAP CoAPConfig `yaml:"coap"`
}

func (c *APIConfig) Validate() error {
	if err := c.CoAP.Validate(); err != nil {
		return fmt.Errorf("coap.%w", err)
	}
	return nil
}

type Config struct {
	API APIConfig `yaml:"apis"`
}

func (c *Config) Validate() error {
	if err := c.API.Validate(); err != nil {
		return fmt.Errorf("api.%w", err)
	}
	return nil
}
