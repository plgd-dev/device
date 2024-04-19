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

package device

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/plgd-dev/device/v2/bridge/device/cloud"
	"github.com/plgd-dev/device/v2/bridge/device/credential"
	"github.com/plgd-dev/device/v2/pkg/eventloop"
	"github.com/plgd-dev/device/v2/pkg/log"
	"github.com/plgd-dev/device/v2/schema"
	"github.com/stretchr/testify/require"
	wotTD "github.com/web-of-things-open-source/thingdescription-go/thingDescription"
)

func TestOptions(t *testing.T) {
	cfg := OptionsCfg{}

	opts := []Option{}
	onDeviceUpdated := func(*Device) {
		// no-op
	}
	opts = append(opts, WithOnDeviceUpdated(onDeviceUpdated))

	getAdditionalPropertiesForResponseFunc := func() map[string]interface{} {
		return nil
	}
	opts = append(opts, WithGetAdditionalPropertiesForResponse(getAdditionalPropertiesForResponseFunc))

	getCertificates := func(string) []tls.Certificate {
		return nil
	}
	opts = append(opts, WithGetCertificates(getCertificates))

	getCAPool := func() []*x509.Certificate {
		return []*x509.Certificate{{}}
	}
	caPool := credential.MakeCAPool(nil, getCAPool)
	opts = append(opts, WithCAPool(caPool))

	logger := log.NewNilLogger()
	opts = append(opts, WithLogger(logger))

	loop := eventloop.New()
	opts = append(opts, WithEventLoop(loop))

	cloudOpt := cloud.WithTickInterval(time.Second)
	opts = append(opts, WithCloudOptions(cloudOpt))

	getThingDescription := func(context.Context, *Device, schema.Endpoints) *wotTD.ThingDescription {
		return nil
	}
	opts = append(opts, WithThingDescription(getThingDescription))

	for _, o := range opts {
		o(&cfg)
	}

	require.NotNil(t, cfg.onDeviceUpdated)
	require.NotNil(t, cfg.getAdditionalProperties)
	require.NotNil(t, cfg.getCertificates)
	require.NotNil(t, cfg.caPool)
	require.Equal(t, logger, cfg.logger)
	require.Equal(t, loop, cfg.loop)
	require.Len(t, cfg.cloudOptions, 1)
	require.NotNil(t, cfg.getThingDescription)
}
