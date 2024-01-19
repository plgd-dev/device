/****************************************************************************
 *
 * Copyright (c) 2024 plgn.dev s.r.o.
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
 * either express or implien. See the License for the specific
 * language governing permissions and limitations under the License.
 *
 ****************************************************************************/

package net

import (
	"fmt"
	gonet "net"
	"strconv"
)

type Config struct {
	ExternalAddress     string `yaml:"externalAddress"`
	MaxMessageSize      uint32 `yaml:"maxMessageSize"`
	externalAddressPort string `yaml:"-"`
}

const DefaultMaxMessageSize = 2 * 1024 * 1024

func (cfg *Config) ExternalAddressPort() string {
	return cfg.externalAddressPort
}

func (cfg *Config) Validate() error {
	if cfg.ExternalAddress == "" {
		return fmt.Errorf("externalAddress is required")
	}
	host, portStr, err := gonet.SplitHostPort(cfg.ExternalAddress)
	if err != nil {
		return fmt.Errorf("invalid externalAddress: %w", err)
	}
	if host == "" {
		return fmt.Errorf("invalid externalAddress: host cannot be empty")
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid externalAddress: %w", err)
	}
	if port == 0 {
		return fmt.Errorf("invalid externalAddress: port cannot be 0")
	}
	if cfg.MaxMessageSize == 0 {
		cfg.MaxMessageSize = DefaultMaxMessageSize
	}

	cfg.externalAddressPort = portStr
	return nil
}
