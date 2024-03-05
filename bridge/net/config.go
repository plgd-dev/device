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
	"time"

	"github.com/hashicorp/go-multierror"
)

const (
	UDP4 = "udp4"
	UDP6 = "udp6"
)

var ErrInvalidExternalAddress = fmt.Errorf("invalid externalAddress")

type externalAddressPort struct {
	host    string
	port    string
	network string
}

type externalAddressesPort []externalAddressPort

func (extAddresses externalAddressesPort) filterByNetwork(network string) externalAddressesPort {
	var filtered externalAddressesPort
	for _, e := range extAddresses {
		if e.network == network {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (extAddresses externalAddressesPort) filterByPort(port string) externalAddressesPort {
	var filtered externalAddressesPort
	for _, e := range extAddresses {
		if e.port == port {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

type Config struct {
	ExternalAddresses     []string              `yaml:"externalAddresses"`
	MaxMessageSize        uint32                `yaml:"maxMessageSize"`
	DeduplicationLifetime time.Duration         `yaml:"deduplicationLifetime"`
	externalAddressesPort externalAddressesPort `yaml:"-"`
}

const DefaultMaxMessageSize = 2 * 1024 * 1024

func errInvalidExternalAddress(err error) error {
	return fmt.Errorf("%w: %w", ErrInvalidExternalAddress, err)
}

func validateExternalAddress(addr string) (externalAddressPort, error) {
	host, portStr, err := gonet.SplitHostPort(addr)
	if err != nil {
		return externalAddressPort{}, errInvalidExternalAddress(err)
	}
	if host == "" {
		return externalAddressPort{}, errInvalidExternalAddress(fmt.Errorf("host cannot be empty"))
	}
	_, err = strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return externalAddressPort{}, errInvalidExternalAddress(err)
	}

	_, errIpv4 := gonet.ResolveUDPAddr(UDP4, addr)
	_, errIpv6 := gonet.ResolveUDPAddr(UDP6, addr)
	if errIpv4 != nil && errIpv6 != nil {
		var errs *multierror.Error
		errs = multierror.Append(errs, errIpv4)
		errs = multierror.Append(errs, errIpv6)
		return externalAddressPort{}, fmt.Errorf("%w: %w", ErrInvalidExternalAddress, errs)
	}
	network := UDP6
	if errIpv4 == nil {
		network = UDP4
	}

	return externalAddressPort{
		host:    host,
		port:    portStr,
		network: network,
	}, nil
}

func (cfg *Config) Validate() error {
	if len(cfg.ExternalAddresses) == 0 {
		return fmt.Errorf("%w: cannot be empty", ErrInvalidExternalAddress)
	}
	if cfg.MaxMessageSize == 0 {
		cfg.MaxMessageSize = DefaultMaxMessageSize
	}
	if cfg.DeduplicationLifetime == 0 {
		cfg.DeduplicationLifetime = 8 * time.Second
	}
	externalAddressesPort := make([]externalAddressPort, 0, len(cfg.ExternalAddresses))
	for i, e := range cfg.ExternalAddresses {
		extAddress, err := validateExternalAddress(e)
		if err != nil {
			return fmt.Errorf("invalid configuration [%v:%v]: %w", i, e, err)
		}
		externalAddressesPort = append(externalAddressesPort, extAddress)
	}
	cfg.externalAddressesPort = externalAddressesPort
	return nil
}
