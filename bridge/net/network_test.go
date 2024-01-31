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

package net_test

import (
	"net"
	"testing"

	bridgeNet "github.com/plgd-dev/device/v2/bridge/net"
	"github.com/stretchr/testify/require"
)

func TestGetPortFromAddress(t *testing.T) {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:42")
	require.NoError(t, err)
	port, err := bridgeNet.GetPortFromAddress(udpAddr)
	require.NoError(t, err)
	require.Equal(t, "42", port)

	tcpAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:42")
	require.NoError(t, err)
	port, err = bridgeNet.GetPortFromAddress(tcpAddr)
	require.NoError(t, err)
	require.Equal(t, "42", port)
}

func TestGetPortFromAddress_Fail(t *testing.T) {
	ipAddr, err := net.ResolveIPAddr("ip", "127.0.0.1")
	require.NoError(t, err)
	_, err = bridgeNet.GetPortFromAddress(ipAddr)
	require.Error(t, err)
}
