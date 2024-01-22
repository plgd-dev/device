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

package maintenance_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/plgd-dev/device/v2/bridge/net"
	"github.com/plgd-dev/device/v2/bridge/resources/maintenance"
	"github.com/plgd-dev/device/v2/pkg/codec/cbor"
	maintenanceSchema "github.com/plgd-dev/device/v2/schema/maintenance"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/stretchr/testify/require"
)

func TestMaintenanceGet(t *testing.T) {
	mnt := maintenance.New(maintenanceSchema.ResourceURI, func() {})
	require.NotNil(t, mnt)

	req := pool.NewMessage(context.Background())
	req.SetContentFormat(message.AppOcfCbor)
	resp, err := mnt.Get(&net.Request{
		Message: req,
	})
	require.NoError(t, err)

	var mntData maintenanceSchema.Maintenance
	err = cbor.ReadFrom(resp.Body(), &mntData)
	require.NoError(t, err)
	require.False(t, mntData.FactoryReset)
}

func TestMaintenancePost(t *testing.T) {
	invoked := false
	mnt := maintenance.New(maintenanceSchema.ResourceURI, func() {
		invoked = true
	})
	require.NotNil(t, mnt)

	reqInvalid := pool.NewMessage(context.Background())
	reqInvalid.SetContentFormat(message.TextPlain)
	reqInvalid.SetBody(bytes.NewReader([]byte("")))
	resp, err := mnt.Post(&net.Request{
		Message: reqInvalid,
	})
	require.NoError(t, err)
	require.Equal(t, codes.BadRequest, resp.Code())
	require.False(t, invoked)

	d, err := cbor.Encode(maintenanceSchema.MaintenanceUpdateRequest{
		FactoryReset: true,
	})
	require.NoError(t, err)
	req := pool.NewMessage(context.Background())
	req.SetContentFormat(message.AppOcfCbor)
	req.SetBody(bytes.NewReader(d))
	resp, err = mnt.Post(&net.Request{
		Message: req,
	})
	require.NoError(t, err)
	require.Equal(t, codes.Changed, resp.Code())
	require.True(t, invoked)
	var mntData maintenanceSchema.Maintenance
	err = cbor.ReadFrom(resp.Body(), &mntData)
	require.NoError(t, err)
	require.False(t, mntData.FactoryReset)
}
