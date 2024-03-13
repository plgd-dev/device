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

package test

import (
	"context"
	"crypto/tls"
	"sync"
	"testing"

	"github.com/plgd-dev/device/v2/test"
	"github.com/plgd-dev/device/v2/test/coap-gateway/service"
	"github.com/stretchr/testify/require"
)

const (
	COAP_GW_HOST = "localhost:21002"
)

func MakeConfig(t *testing.T) service.Config {
	return service.Config{
		Addr: COAP_GW_HOST,
		TLS: service.TLSConfig{
			Enabled: true,
			Config: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec
				MinVersion:         tls.VersionTLS12,
				Certificates:       []tls.Certificate{test.GetCoapCertificate(t)},
			},
		},
	}
}

func New(t *testing.T, getHandler service.GetServiceHandler, onShutdown service.OnShutdown) func() {
	ctx := context.Background()
	s, err := service.New(ctx, MakeConfig(t), getHandler)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = s.Serve()
	}()
	return func() {
		_ = s.Close()
		wg.Wait()
		if onShutdown != nil {
			for _, c := range s.GetClients() {
				onShutdown(c.GetServiceHandler())
			}
		}
	}
}
