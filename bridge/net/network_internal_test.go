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
	"net"
	"testing"

	"github.com/plgd-dev/device/v2/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestGetPortFromAddress(t *testing.T) {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:42")
	require.NoError(t, err)
	port, err := getPortFromAddress(udpAddr)
	require.NoError(t, err)
	require.Equal(t, "42", port)

	tcpAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:42")
	require.NoError(t, err)
	port, err = getPortFromAddress(tcpAddr)
	require.NoError(t, err)
	require.Equal(t, "42", port)
}

func TestGetPortFromAddress_Fail(t *testing.T) {
	ipAddr, err := net.ResolveIPAddr("ip", "127.0.0.1")
	require.NoError(t, err)
	_, err = getPortFromAddress(ipAddr)
	require.Error(t, err)
}

func TestGetNetwork(t *testing.T) {
	cfg := Config{
		ExternalAddresses: []string{"127.0.0.1:42", "[::1]:42", "127.0.0.1:13", "[::1]:37"},
	}
	err := cfg.Validate()
	require.NoError(t, err)

	n := &Net{
		cfg: cfg,
	}

	network := n.getNetwork(nil, "127.0.0.1", "42")
	require.Equal(t, UDP4, network)
	network = n.getNetwork(nil, "[::1]", "42")
	require.Equal(t, UDP6, network)
	network = n.getNetwork(nil, "127.0.0.1", "13")
	require.Equal(t, UDP4, network)
	network = n.getNetwork(nil, "[::1]", "37")
	require.Equal(t, UDP6, network)
}

func TestClose(t *testing.T) {
	cfg := Config{
		ExternalAddresses: []string{"127.0.0.1:0"},
	}
	n, err := New(cfg, nil, log.NewNilLogger())
	require.NoError(t, err)

	require.NotEqual(t, 0, n.cfg.externalAddressesPort[0])
	checkUDPPort := func(opened bool) {
		listenAddress, errL := net.ResolveUDPAddr(n.cfg.externalAddressesPort[0].network, ":"+n.cfg.externalAddressesPort[0].port)
		require.NoError(t, errL)
		conn, errL := net.ListenUDP(n.cfg.externalAddressesPort[0].network, listenAddress)
		if opened {
			require.Error(t, errL)
			return
		}
		require.NoError(t, errL)
		errL = conn.Close()
		require.NoError(t, errL)
	}
	checkUDPPort(true)

	err = n.Close()
	require.NoError(t, err)
	checkUDPPort(false)
}
